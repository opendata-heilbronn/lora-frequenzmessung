package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/opendata-heilbronn/lora-frequenzmessung/Share/pwhash"
)

var (
	ErrUserNotFound            = errors.New("user not found")
	ErrSessionAboveMaxLifetime = errors.New("session has reached its max lifetime")
)

const (
	blindHash          = "$argon2id$v=19$m=65536,t=1,p=10$U3VWanh0SFI0SGxkaXBpRw$gWWS41rZyDRb6uGEVHU/Pwis3iB2AIQOHF2uJ6tSjZU"
	refreshTokenCookie = "refresh_token"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	AccessToken string `json:"token"`
}

type AuthRepo interface {
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	GetUserByID(ctx context.Context, id string) (*User, error)
	UpdateUser(ctx context.Context, user User, fieldMask []string) error
}

type User struct {
	Id           string
	Username     string
	PasswordHash *string
}

func loginHandler(repo AuthRepo) fiber.Handler {
	secret := os.Getenv("JWT_SECRET")

	return func(c fiber.Ctx) error {
		var req LoginRequest
		if err := c.Bind().JSON(&req); err != nil {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
		}

		user, err := repo.GetUserByUsername(c.Context(), req.Username)
		if errors.Is(err, ErrUserNotFound) {
			// blind comparison to prevent timing attacks for user enumeration
			_, _ = pwhash.Verify(req.Password, blindHash)

			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "invalid username or password"})
		}
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
		}

		if user.PasswordHash == nil {
			// blind comparison to prevent timing attacks for user enumeration
			_, _ = pwhash.Verify(req.Password, blindHash)

			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "invalid username or password"})
		}

		updatedHash, ok := pwhash.Verify(req.Password, *user.PasswordHash)
		if !ok {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "invalid username or password"})
		}

		if updatedHash != "" {
			user.PasswordHash = new(updatedHash)

			err = repo.UpdateUser(c.Context(), *user, []string{"password"})
			if err != nil {
				return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
			}
		}

		accessToken, refreshToken, err := generateTokens(*user, secret, nil)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
		}

		c.Cookie(&fiber.Cookie{
			Expires:  time.Now().Add(24*time.Hour - 1*time.Minute),
			Name:     refreshTokenCookie,
			Value:    refreshToken,
			Path:     "/auth/refresh",
			SameSite: fiber.CookieSameSiteLaxMode,
			Secure:   true,
			HTTPOnly: true,
		})

		return c.JSON(LoginResponse{
			AccessToken: accessToken,
		})
	}
}

func refreshHandler(repo AuthRepo) fiber.Handler {
	secret := os.Getenv("JWT_SECRET")

	return func(ctx fiber.Ctx) error {
		refreshToken := ctx.Cookies(refreshTokenCookie, "")
		if refreshToken == "" {
			ctx.Status(http.StatusUnauthorized)
			return ctx.End()
		}

		var refreshClaims jwt.MapClaims
		_, err := jwt.ParseWithClaims(refreshToken, &refreshClaims, func(token *jwt.Token) (any, error) {
			return []byte(secret), nil
		}, jwt.WithExpirationRequired())
		if err != nil {
			ctx.Status(http.StatusUnauthorized)
			return ctx.End()
		}

		userId, ok := refreshClaims["sub"].(string)
		if !ok {
			ctx.Status(http.StatusUnauthorized)
			return ctx.End()
		}

		sessionStartUnix, ok := refreshClaims["session_start"].(float64)
		if !ok {
			ctx.Status(http.StatusUnauthorized)
			return ctx.End()
		}

		user, err := repo.GetUserByID(ctx, userId)
		if errors.Is(err, ErrUserNotFound) {
			ctx.Status(http.StatusUnauthorized)
			return ctx.End()
		}
		if err != nil {
			ctx.Status(http.StatusInternalServerError)
			return ctx.End()
		}

		accessToken, newRefreshToken, err := generateTokens(*user, secret, new(time.Unix(int64(sessionStartUnix), 0)))
		if errors.Is(err, ErrSessionAboveMaxLifetime) {
			ctx.Status(http.StatusUnauthorized)
			return ctx.End()
		}
		if err != nil {
			ctx.Status(http.StatusInternalServerError)
			return ctx.End()
		}

		ctx.Cookie(&fiber.Cookie{
			Expires:  time.Now().Add(24*time.Hour - 1*time.Minute),
			Name:     refreshTokenCookie,
			Value:    newRefreshToken,
			Path:     "/auth/refresh",
			SameSite: fiber.CookieSameSiteLaxMode,
			Secure:   true,
			HTTPOnly: true,
		})

		return ctx.JSON(LoginResponse{
			AccessToken: accessToken,
		})
	}
}

func generateTokens(user User, secret string, sessionStart *time.Time) (string, string, error) {
	now := time.Now()

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":      user.Id,
		"username": user.Username,
		"exp":      jwt.NewNumericDate(now.Add(15 * time.Minute)),
		"iat":      jwt.NewNumericDate(now),
	})

	sessionStartTime := now
	if sessionStart != nil {
		sessionStartTime = *sessionStart
	}

	refreshTokenMaxExp := sessionStartTime.Add(7 * 24 * time.Hour)
	if time.Now().After(refreshTokenMaxExp) {
		return "", "", ErrSessionAboveMaxLifetime
	}

	refreshTokenExp := now.Add(24 * time.Hour)
	if refreshTokenExp.After(refreshTokenMaxExp) {
		refreshTokenExp = refreshTokenMaxExp
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":           user.Id,
		"exp":           jwt.NewNumericDate(refreshTokenExp),
		"iat":           jwt.NewNumericDate(now),
		"session_start": jwt.NewNumericDate(sessionStartTime),
	})

	tokenString, err := accessToken.SignedString([]byte(secret))
	if err != nil {
		return "", "", err
	}

	refreshTokenString, err := refreshToken.SignedString([]byte(secret))
	if err != nil {
		return "", "", err
	}

	return tokenString, refreshTokenString, nil
}
