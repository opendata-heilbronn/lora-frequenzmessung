package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5"
)

type Sensor struct {
	ID          int       `json:"id"`
	UUID        string    `json:"uuid"`
	Name        string    `json:"name"`
	Longitude   float64   `json:"longitude"`
	Latitude    float64   `json:"latitude"`
	Type        string    `json:"type"`
	DevEUI      string    `json:"dev_eui"`
	AppKey      string    `json:"app_key"`
	TtnDeviceID string    `json:"ttn_device_id"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreateSensorRequest struct {
	Name      string  `json:"name"`
	Longitude float64 `json:"longitude"`
	Latitude  float64 `json:"latitude"`
}

func getSensors(conn *pgx.Conn, ctx context.Context) fiber.Handler {
	return func(c fiber.Ctx) error {
		rows, err := conn.Query(ctx, `
			SELECT id, uuid, name, longitude, latitude, type,
			       COALESCE(dev_eui,''), COALESCE(app_key,''),
			       COALESCE(ttn_device_id,''), created_at
			FROM sensors
			ORDER BY created_at DESC`)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		defer rows.Close()

		sensors := []Sensor{}
		for rows.Next() {
			var s Sensor
			err := rows.Scan(&s.ID, &s.UUID, &s.Name, &s.Longitude, &s.Latitude,
				&s.Type, &s.DevEUI, &s.AppKey, &s.TtnDeviceID, &s.CreatedAt)
			if err != nil {
				return c.Status(500).JSON(fiber.Map{"error": err.Error()})
			}
			sensors = append(sensors, s)
		}
		return c.JSON(sensors)
	}
}

func createSensor(conn *pgx.Conn, ctx context.Context) fiber.Handler {
	return func(c fiber.Ctx) error {
		var req CreateSensorRequest
		if err := c.Bind().JSON(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		if req.Name == "" {
			return c.Status(400).JSON(fiber.Map{"error": "name is required"})
		}

		// Generate sensor_id (8-char hex = 4 random bytes)
		idBytes := make([]byte, 4)
		rand.Read(idBytes)
		sensorID := hex.EncodeToString(idBytes)

		_, err := conn.Exec(ctx,
			`INSERT INTO sensors (uuid, name, longitude, latitude, type)
			 VALUES ($1, $2, $3, $4, $5)`,
			sensorID, req.Name, req.Longitude, req.Latitude, "density")
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		return c.Status(201).JSON(fiber.Map{
			"uuid":      sensorID,
			"name":      req.Name,
			"longitude": req.Longitude,
			"latitude":  req.Latitude,
			"type":      "density",
		})
	}
}

func registerTTN(conn *pgx.Conn, ctx context.Context) fiber.Handler {
	return func(c fiber.Ctx) error {
		uuid := c.Params("uuid")

		var devEUI string
		err := conn.QueryRow(ctx,
			`SELECT COALESCE(dev_eui,'') FROM sensors WHERE uuid = $1`, uuid).
			Scan(&devEUI)
		if err == pgx.ErrNoRows {
			return c.Status(404).JSON(fiber.Map{"error": "sensor not found"})
		}
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		if devEUI != "" {
			return c.Status(409).JSON(fiber.Map{"error": "sensor already linked to TTN"})
		}

		// Generate devEUI (8 bytes → 16-char uppercase hex)
		devEuiBytes := make([]byte, 8)
		rand.Read(devEuiBytes)
		newDevEUI := fmt.Sprintf("%X", devEuiBytes)

		// Generate appKey (16 bytes → 32-char uppercase hex)
		appKeyBytes := make([]byte, 16)
		rand.Read(appKeyBytes)
		appKey := fmt.Sprintf("%X", appKeyBytes)

		ttnDeviceID := fmt.Sprintf("sensor-%s", uuid)

		// Register in TTN — if it fails, don't save partial credentials
		if err := registerTTNDevice(ttnDeviceID, newDevEUI, appKey); err != nil {
			return c.Status(502).JSON(fiber.Map{"error": fmt.Sprintf("TTN registration failed: %v", err)})
		}

		tag, err := conn.Exec(ctx,
			`UPDATE sensors SET dev_eui = $1, app_key = $2, ttn_device_id = $3 WHERE uuid = $4`,
			newDevEUI, appKey, ttnDeviceID, uuid)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		if tag.RowsAffected() == 0 {
			return c.Status(404).JSON(fiber.Map{"error": "sensor not found"})
		}

		return c.JSON(fiber.Map{
			"dev_eui":       newDevEUI,
			"app_key":       appKey,
			"ttn_device_id": ttnDeviceID,
		})
	}
}

func deleteSensor(conn *pgx.Conn, ctx context.Context) fiber.Handler {
	return func(c fiber.Ctx) error {
		uuid := c.Params("uuid")

		var ttnDeviceID string
		err := conn.QueryRow(ctx,
			`SELECT COALESCE(ttn_device_id,'') FROM sensors WHERE uuid = $1`, uuid).
			Scan(&ttnDeviceID)
		if err == pgx.ErrNoRows {
			return c.Status(404).JSON(fiber.Map{"error": "sensor not found"})
		}
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		// Delete from TTN (best effort)
		if ttnDeviceID != "" {
			deleteTTNDevice(ttnDeviceID)
		}

		_, err = conn.Exec(ctx, `DELETE FROM sensors WHERE uuid = $1`, uuid)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		return c.Status(204).SendString("")
	}
}
