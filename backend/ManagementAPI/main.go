package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"embed"
	"encoding/base64"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/opendata-heilbronn/lora-frequenzmessung/Share/database"
	"github.com/opendata-heilbronn/lora-frequenzmessung/Share/pwhash"
)

//go:embed migrations/*.sql
var migrations embed.FS

// uuidRegex validates sensor UUIDs (8-char lowercase hex).
var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}$`)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	// Start scheduled firmware version polling
	intervalHours := 24
	if v := os.Getenv("VERSION_CHECK_INTERVAL_HOURS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			intervalHours = n
		}
	}
	StartVersionCheckScheduler(time.Duration(intervalHours) * time.Hour)

	dbURI := os.Getenv("DATABASE_URI")
	if dbURI == "" {
		dbURI = "postgres://management:management@localhost:5432/management?sslmode=disable"
	}

	db, err := database.Connect(database.Config{
		URI:          dbURI,
		MaxOpenConns: 8,
	})
	if err != nil {
		logger.Error("connecting to db", slog.String("error", err.Error()))
		return
	}

	err = database.Migrate(context.Background(), db, logger, migrations)
	if err != nil {
		logger.Error("migrating db", slog.String("error", err.Error()))
		return
	}

	initialUserPassword, err := ensureInitialUserExists(context.Background(), db)
	if err != nil {
		logger.Error("creating initial user", slog.String("error", err.Error()))
		return
	}

	if initialUserPassword != "" {
		logger.Info("initial user created", slog.String("username", "admin"), slog.String("password", initialUserPassword))
	}

	app := fiber.New()
	userRepo := &PostgresUserRepo{DB: db}

	// CORS — read allowed origins from env, with sensible defaults
	allowedOrigins := os.Getenv("CORS_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "http://localhost:5173,http://localhost:8080"
	}
	app.Use(cors.New(cors.Config{
		AllowOrigins: strings.Split(allowedOrigins, ","),
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Content-Type", "Authorization"},
	}))

	// Public routes
	app.Post("/auth/login", loginHandler(userRepo))
	app.Post("/auth/refresh", refreshHandler(userRepo))
	app.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
	app.Get("/version", func(c fiber.Ctx) error {
		version := os.Getenv("IMAGE_VERSION")
		if version == "" {
			version = "dev"
		}
		sha := os.Getenv("GIT_SHA")
		if sha == "" {
			sha = "unknown"
		}
		run := os.Getenv("RUN_NUMBER")
		if run == "" {
			run = "0"
		}
		// Derive branch from version prefix (before the first '-')
		branch := "unknown"
		if len(version) > 0 {
			parts := strings.SplitN(version, "-", 2)
			if parts[0] == "latest" {
				branch = "main"
			} else if parts[0] == "develop" {
				branch = "develop"
			}
		}
		return c.JSON(fiber.Map{
			"version": version,
			"sha":     sha,
			"run":     run,
			"branch":  branch,
		})
	})
	// Public firmware flashing endpoints (no auth) — required by esp-web-tools
	app.Get("/api/sensors/:uuid/manifest.json", getManifestHandler)
	app.Get("/api/sensors/:uuid/bootloader.bin", getBootloaderBinHandler)
	app.Get("/api/sensors/:uuid/partitions.bin", getPartitionsBinHandler)
	app.Get("/api/sensors/:uuid/otadata.bin", getOtaDataBinHandler)
	app.Get("/api/sensors/:uuid/firmware.bin", getFirmwareBinHandler)

	// Protected API routes
	api := app.Group("/api", jwtMiddleware)
	api.Get("/sensors", proxySensors)
	api.Post("/sensors", proxyCreateSensor)
	api.Get("/sensors/:uuid", proxyGetSensor)
	api.Delete("/sensors/:uuid", proxyDeleteSensor)
	api.Post("/sensors/:uuid/register-ttn", registerTTNHandler)
	api.Post("/sensors/:uuid/build-firmware", buildFirmwareHandler)
	api.Get("/sensors/:uuid/build-status", getBuildStatusHandler)
	api.Post("/sensors/:uuid/request-version", requestVersionHandler)
	api.Post("/sensors/request-all-versions", requestAllVersionsHandler)
	api.Get("/provision-config", getProvisionConfigHandler)
	api.Post("/sensors/:uuid/trigger-ota", triggerOTAHandler)
	api.Get("/users", listUsers(userRepo))
	api.Post("/users", createUser(userRepo))
	api.Patch("/users/:id", updateUser(userRepo))
	api.Delete("/users/:id", deleteUser(userRepo))

	// Graceful shutdown
	go func() {
		if err := app.Listen(":3002"); err != nil {
			log.Printf("ManagementAPI error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down ManagementAPI...")
	if err := app.ShutdownWithTimeout(10 * time.Second); err != nil {
		log.Printf("Server forced shutdown: %v", err)
	}
	log.Println("ManagementAPI stopped")
}

func ensureInitialUserExists(ctx context.Context, db *sql.DB) (string, error) {
	var usercount int64

	err := db.QueryRowContext(ctx, `select count(*) from users`).Scan(&usercount)
	if err != nil {
		return "", err
	}

	if usercount != 0 {
		return "", nil
	}

	pwBytes := make([]byte, 32)
	_, err = rand.Read(pwBytes)
	if err != nil {
		return "", err
	}

	pwString := base64.RawStdEncoding.EncodeToString(pwBytes)
	pwHash, err := pwhash.Create(pwString)
	if err != nil {
		return "", err
	}

	_, err = db.ExecContext(ctx, `insert into users (username, password_hash) values ($1, $2)`, "admin", pwHash)
	if err != nil {
		return "", err
	}

	return pwString, nil
}
