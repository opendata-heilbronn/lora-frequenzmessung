package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"regexp"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/opendata-heilbronn/lora-frequenzmessung/Share/Misc"
	structs2 "github.com/opendata-heilbronn/lora-frequenzmessung/structs"
)

// uuidRegex validates sensor UUIDs (8-char lowercase hex).
var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}$`)

func validateUUID(c fiber.Ctx) (string, error) {
	uuid := c.Params("uuid")
	if !uuidRegex.MatchString(uuid) {
		return "", c.Status(400).JSON(fiber.Map{"error": "invalid sensor UUID"})
	}
	return uuid, nil
}

func main() {
	Misc.StartUp()
	log.Println("STARTING Backend")

	app := fiber.New()
	ctx := context.Background()
	DBDns := Misc.GetDBDsn()
	pool, err := pgxpool.New(ctx, DBDns)
	if err != nil {
		fmt.Fprintf(os.Stderr, " database: %v\n", DBDns)
		fmt.Println("------------------------------------")
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	runMigrations(DBDns)

	// X-Internal-Key authentication — this service is internal-only
	internalKey := Misc.GetInternalAPIKey()
	app.Use(func(c fiber.Ctx) error {
		if c.Get("X-Internal-Key") != internalKey {
			return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
		}
		return c.Next()
	})

	// Data ingestion
	app.Post("/internal/sensor-data", func(c fiber.Ctx) error {
		p := new(structs2.DensityDataWithClient)
		err := c.AutoFormat(p)
		if err != nil {
			return err
		}
		err = json.Unmarshal(c.Body(), p)
		if err != nil {
			return err
		}

		if err := sendData(
			pool,
			ctx,
			p.Data.SensorID,
			p.Client.Name,
			p.Client.Longitude,
			p.Client.Latitude,
			p.Data.Value,
			p.DataType); err != nil {
			log.Printf("ERROR: sendData failed: %v", err)
			return c.Status(500).JSON(fiber.Map{"error": "failed to insert sensor data"})
		}

		return c.Status(fiber.StatusAccepted).SendString("Message accepted")
	})

	// Sensor CRUD
	app.Get("/internal/sensors", getSensors(pool, ctx))
	app.Post("/internal/sensors", createSensor(pool, ctx))
	app.Get("/internal/sensors/:uuid", getSensor(pool, ctx))
	app.Delete("/internal/sensors/:uuid", deleteSensor(pool, ctx))
	app.Put("/internal/sensors/:uuid/ttn", updateSensorTTN(pool, ctx))
	app.Patch("/internal/sensors/:uuid/firmware-version", updateFirmwareVersion(pool, ctx))

	// Graceful shutdown
	go func() {
		if err := app.Listen(":3001"); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")
	if err := app.ShutdownWithTimeout(10 * time.Second); err != nil {
		log.Printf("Server forced shutdown: %v", err)
	}
	log.Println("Server stopped")
}

func sendData(pool *pgxpool.Pool, ctx context.Context, uuid string, sensorName string, longitude float64, latitude float64, value float64, sensorType string) error {
	t := time.Now()
	queryInsertMetadata := `INSERT INTO sensor_data (
                        sensor_id,
                        name,
                        time,
                        longitude,
                        latitude,
                        value,
                        type
                        ) VALUES ($1, $2,$3,$4,$5,$6,$7);`

	_, err := pool.Exec(ctx, queryInsertMetadata, uuid, sensorName, t, longitude, latitude, value, sensorType)
	if err != nil {
		return fmt.Errorf("unable to insert data into database: %w", err)
	}
	return nil
}
