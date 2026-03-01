package main

import (
	"log"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
)

// uuidRegex validates sensor UUIDs (8-char lowercase hex).
var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}$`)

func main() {
	// Start scheduled firmware version polling
	intervalHours := 24
	if v := os.Getenv("VERSION_CHECK_INTERVAL_HOURS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			intervalHours = n
		}
	}
	StartVersionCheckScheduler(time.Duration(intervalHours) * time.Hour)

	app := fiber.New()

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
	app.Post("/auth/login", loginHandler)
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
