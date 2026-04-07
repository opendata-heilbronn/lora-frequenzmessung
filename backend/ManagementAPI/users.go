package main

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/opendata-heilbronn/lora-frequenzmessung/Share/pwhash"
)

type UserRepo interface {
	ListUsers(ctx context.Context, pageSize int64, lastUsername string) ([]User, error)
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	GetUserByID(ctx context.Context, id string) (*User, error)
	CreateUser(ctx context.Context, user User) error
	UpdateUser(ctx context.Context, user User, fieldMask []string) error
	DeleteUser(ctx context.Context, id string) error
}

type ApiUser struct {
	Id          string `json:"id"`
	Username    string `json:"username"`
	HasPassword bool   `json:"has_password"`
}

type ListUsersResponse struct {
	Users         []ApiUser `json:"users"`
	NextPageToken string    `json:"next_page_token"`
}

func listUsers(repo UserRepo) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		pageSizeStr := ctx.Query("page_size", "100")
		pageSize, err := strconv.ParseInt(pageSizeStr, 10, 32)
		if err != nil {
			return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "page_size must be a number"})
		}

		var lastUsername string

		pageToken := ctx.Query("page_token", "")
		if pageToken != "" {
			decoded, err := base64.RawURLEncoding.DecodeString(pageToken)
			if err != nil {
				return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid page_token"})
			}

			lastUsername = string(decoded)
		}

		users, err := repo.ListUsers(ctx, pageSize+1, lastUsername)
		if err != nil {
			return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
		}

		nextPageToken := ""
		if int64(len(users)) > pageSize {
			users = users[:pageSize]
			nextPageToken = base64.RawURLEncoding.EncodeToString([]byte(users[pageSize-1].Username))
		}

		response := ListUsersResponse{
			Users:         make([]ApiUser, 0, pageSize),
			NextPageToken: nextPageToken,
		}

		for _, user := range users {
			response.Users = append(response.Users, ApiUser{
				Id:          user.Id,
				Username:    user.Username,
				HasPassword: user.PasswordHash != nil,
			})
		}

		return ctx.JSON(response)
	}
}

type CreateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

var validUsernameRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9._-]*$`)

func createUser(repo UserRepo) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		var createUserRequest CreateUserRequest

		err := ctx.Bind().JSON(&createUserRequest)
		if err != nil {
			return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		if !validUsernameRegex.MatchString(createUserRequest.Username) {
			return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid characters in username. Valid characters: alphanumeric, underscore and minus. Must start with an alpha character"})
		}

		var passwordHash *string
		if createUserRequest.Password != "" {
			pwHash, err := pwhash.Create(createUserRequest.Password)
			if err != nil {
				return err
			}

			passwordHash = &pwHash
		}

		err = repo.CreateUser(ctx, User{
			Username:     createUserRequest.Username,
			PasswordHash: passwordHash,
		})
		if err != nil {
			return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
		}

		newUser, err := repo.GetUserByUsername(ctx, createUserRequest.Username)
		if err != nil {
			return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
		}

		return ctx.JSON(ApiUser{
			Id:          newUser.Id,
			Username:    newUser.Username,
			HasPassword: newUser.PasswordHash != nil,
		})
	}
}

type UpdateUserRequest struct {
	Username string
	Password string
}

func updateUser(repo UserRepo) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		id := ctx.Params("id")
		if _, err := uuid.Parse(id); err != nil {
			return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "user id must be a valid UUID"})
		}

		fieldMaskString := ctx.Query("field_mask")
		fieldMask := strings.Split(fieldMaskString, ",")
		if len(fieldMask) == 0 {
			return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "field_mask must not be empty"})
		}

		var apiUser UpdateUserRequest
		err := ctx.Bind().JSON(&apiUser)
		if err != nil {
			return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		user := User{
			Id: id,
		}

		for _, path := range fieldMask {
			switch path {
			case "username":
				if !validUsernameRegex.MatchString(apiUser.Username) {
					return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid characters in username. Valid characters: alphanumeric, underscore and minus. Must start with an alpha character"})
				}

				user.Username = apiUser.Username
			case "password":
				if apiUser.Password == "" {
					user.PasswordHash = nil
					continue
				}

				hash, err := pwhash.Create(apiUser.Password)
				if err != nil {
					return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
				}

				user.PasswordHash = new(hash)
			default:
				return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "unknown field in field_mask"})
			}
		}

		err = repo.UpdateUser(ctx, user, fieldMask)
		if err != nil {
			return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
		}

		updatedUser, err := repo.GetUserByID(ctx, id)
		if err != nil {
			return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
		}

		return ctx.JSON(ApiUser{
			Id:          updatedUser.Id,
			Username:    updatedUser.Username,
			HasPassword: updatedUser.PasswordHash != nil,
		})
	}
}

func deleteUser(repo UserRepo) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		id := ctx.Params("id")
		if _, err := uuid.Parse(id); err != nil {
			return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "user id must be a valid UUID"})
		}

		err := repo.DeleteUser(ctx, id)
		if err != nil {
			return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
		}

		return ctx.Status(http.StatusOK).End()
	}
}

type PostgresUserRepo struct {
	DB *sql.DB
}

func (p *PostgresUserRepo) ListUsers(ctx context.Context, pageSize int64, lastUsername string) ([]User, error) {
	rows, err := p.DB.QueryContext(ctx, `
		select 
		    id, username, password_hash 
		from users 
		where username > $1 
		order by username 
		limit $2
	`, lastUsername, pageSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]User, 0, pageSize)

	for rows.Next() {
		var user User
		err = rows.Scan(
			&user.Id,
			&user.Username,
			&user.PasswordHash,
		)
		if err != nil {
			return nil, err
		}

		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func (p *PostgresUserRepo) GetUserByID(ctx context.Context, id string) (*User, error) {
	var user User
	err := p.DB.QueryRowContext(ctx, `SELECT id, username, password_hash FROM users WHERE id = $1`, id).Scan(
		&user.Id,
		&user.Username,
		&user.PasswordHash,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (p *PostgresUserRepo) CreateUser(ctx context.Context, user User) error {
	_, err := p.DB.ExecContext(ctx, `insert into users (username, password_hash) values ($1, $2)`, user.Username, user.PasswordHash)
	return err
}

func (p *PostgresUserRepo) UpdateUser(ctx context.Context, user User, fieldMask []string) error {
	query := `update users set `
	args := []any{}

	updateClauses := make([]string, 0, len(fieldMask))

	for i, path := range fieldMask {
		switch path {
		case "username":
			updateClauses = append(updateClauses, fmt.Sprintf(`username = $%d`, i+1))
			args = append(args, user.Username)
		case "password":
			updateClauses = append(updateClauses, fmt.Sprintf("password_hash = $%d", i+1))
			args = append(args, user.PasswordHash)
		}
	}

	query += strings.Join(updateClauses, ", ")
	query += fmt.Sprintf(" where id = $%d", len(fieldMask)+1)
	args = append(args, user.Id)

	_, err := p.DB.ExecContext(ctx, query, args...)
	return err
}

func (p *PostgresUserRepo) DeleteUser(ctx context.Context, id string) error {
	_, err := p.DB.ExecContext(ctx, `delete from users where id = $1`, id)
	return err
}

func (p *PostgresUserRepo) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	var user User
	err := p.DB.QueryRowContext(ctx, `SELECT id, username, password_hash FROM users WHERE username = $1`, username).Scan(
		&user.Id,
		&user.Username,
		&user.PasswordHash,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}
