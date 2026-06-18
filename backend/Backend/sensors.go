package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"math"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Sensor struct {
	ID                  int        `json:"id"`
	UUID                string     `json:"uuid"`
	Name                string     `json:"name"`
	Longitude           float64    `json:"longitude"`
	Latitude            float64    `json:"latitude"`
	Type                string     `json:"type"`
	DevEUI              string     `json:"dev_eui"`
	AppKey              string     `json:"app_key"`
	TtnDeviceID         string     `json:"ttn_device_id"`
	CreatedAt           time.Time  `json:"created_at"`
	LastBatteryValue    *float64   `json:"last_battery_value"`
	LastBatteryTime     *time.Time `json:"last_battery_time"`
	LastDataTime        *time.Time `json:"last_data_time"`
	FirmwareVersion     *string    `json:"firmware_version"`
	FirmwareVersionTime *time.Time `json:"firmware_version_time"`
}

const errSensorNotFound = "sensor not found"

type CreateSensorRequest struct {
	Name      string  `json:"name"`
	Longitude float64 `json:"longitude"`
	Latitude  float64 `json:"latitude"`
}

type UpdateSensorTTNRequest struct {
	DevEUI      string `json:"dev_eui"`
	AppKey      string `json:"app_key"`
	TtnDeviceID string `json:"ttn_device_id"`
}

func getSensors(pool *pgxpool.Pool, ctx context.Context) fiber.Handler {
	return func(c fiber.Ctx) error {
		rows, err := pool.Query(ctx, `
			SELECT s.id, s.uuid, s.name, s.longitude, s.latitude, s.type,
			       COALESCE(s.dev_eui,''), COALESCE(s.app_key,''),
			       COALESCE(s.ttn_device_id,''), s.created_at,
			       batt.value, batt.time,
			       last.time,
			       s.firmware_version, s.firmware_version_time
			FROM sensors s
			LEFT JOIN LATERAL (
			    SELECT value, time
			    FROM sensor_data
			    WHERE sensor_id = s.uuid AND type = 'batteryData'
			    ORDER BY time DESC
			    LIMIT 1
			) batt ON TRUE
			LEFT JOIN LATERAL (
			    SELECT time
			    FROM sensor_data
			    WHERE sensor_id = s.uuid
			    ORDER BY time DESC
			    LIMIT 1
			) last ON TRUE
			ORDER BY s.created_at DESC`)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		defer rows.Close()

		sensors := []Sensor{}

		for rows.Next() {
			var s Sensor

			err := rows.Scan(&s.ID, &s.UUID, &s.Name, &s.Longitude, &s.Latitude,
				&s.Type, &s.DevEUI, &s.AppKey, &s.TtnDeviceID, &s.CreatedAt,
				&s.LastBatteryValue, &s.LastBatteryTime, &s.LastDataTime,
				&s.FirmwareVersion, &s.FirmwareVersionTime)
			if err != nil {
				return c.Status(500).JSON(fiber.Map{"error": err.Error()})
			}

			sensors = append(sensors, s)
		}

		return c.JSON(sensors)
	}
}

func getSensor(pool *pgxpool.Pool, ctx context.Context) fiber.Handler {
	return func(c fiber.Ctx) error {
		uuid, err := validateUUID(c)
		if err != nil {
			return err
		}

		var s Sensor

		err = pool.QueryRow(ctx, `
			SELECT id, uuid, name, longitude, latitude, type,
			       COALESCE(dev_eui,''), COALESCE(app_key,''),
			       COALESCE(ttn_device_id,''), created_at
			FROM sensors WHERE uuid = $1`, uuid).
			Scan(&s.ID, &s.UUID, &s.Name, &s.Longitude, &s.Latitude,
				&s.Type, &s.DevEUI, &s.AppKey, &s.TtnDeviceID, &s.CreatedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return c.Status(404).JSON(fiber.Map{"error": errSensorNotFound})
		}

		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(s)
	}
}

func createSensor(pool *pgxpool.Pool, ctx context.Context) fiber.Handler {
	return func(c fiber.Ctx) error {
		var req CreateSensorRequest
		if err := c.Bind().JSON(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}

		if req.Name == "" || len(req.Name) > 64 {
			return c.Status(400).JSON(fiber.Map{"error": "name is required and must be 1-64 characters"})
		}

		if req.Latitude < -90 || req.Latitude > 90 || math.IsNaN(req.Latitude) || math.IsInf(req.Latitude, 0) {
			return c.Status(400).JSON(fiber.Map{"error": "latitude must be between -90 and 90"})
		}

		if req.Longitude < -180 || req.Longitude > 180 || math.IsNaN(req.Longitude) || math.IsInf(req.Longitude, 0) {
			return c.Status(400).JSON(fiber.Map{"error": "longitude must be between -180 and 180"})
		}

		// Generate sensor_id (8-char hex = 4 random bytes)
		idBytes := make([]byte, 4)
		rand.Read(idBytes)
		sensorID := hex.EncodeToString(idBytes)

		_, err := pool.Exec(ctx,
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

func deleteSensor(pool *pgxpool.Pool, ctx context.Context) fiber.Handler {
	return func(c fiber.Ctx) error {
		uuid, err := validateUUID(c)
		if err != nil {
			return err
		}

		var exists bool

		err = pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM sensors WHERE uuid = $1)`, uuid).
			Scan(&exists)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		if !exists {
			return c.Status(404).JSON(fiber.Map{"error": errSensorNotFound})
		}

		_, err = pool.Exec(ctx, `DELETE FROM sensors WHERE uuid = $1`, uuid)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		return c.Status(204).SendString("")
	}
}

func updateFirmwareVersion(pool *pgxpool.Pool, ctx context.Context) fiber.Handler {
	return func(c fiber.Ctx) error {
		uuid, err := validateUUID(c)
		if err != nil {
			return err
		}

		var req struct {
			Version string `json:"version"`
		}
		if err := c.Bind().JSON(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}

		if req.Version == "" {
			return c.Status(400).JSON(fiber.Map{"error": "version is required"})
		}

		tag, err := pool.Exec(ctx,
			`UPDATE sensors SET firmware_version = $1, firmware_version_time = NOW() WHERE uuid = $2`,
			req.Version, uuid)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		if tag.RowsAffected() == 0 {
			return c.Status(404).JSON(fiber.Map{"error": errSensorNotFound})
		}

		return c.JSON(fiber.Map{"firmware_version": req.Version})
	}
}

func updateSensorTTN(pool *pgxpool.Pool, ctx context.Context) fiber.Handler {
	return func(c fiber.Ctx) error {
		uuid, err := validateUUID(c)
		if err != nil {
			return err
		}

		var req UpdateSensorTTNRequest
		if err := c.Bind().JSON(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}

		tag, err := pool.Exec(ctx,
			`UPDATE sensors SET dev_eui = $1, app_key = $2, ttn_device_id = $3 WHERE uuid = $4`,
			req.DevEUI, req.AppKey, req.TtnDeviceID, uuid)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		if tag.RowsAffected() == 0 {
			return c.Status(404).JSON(fiber.Map{"error": errSensorNotFound})
		}

		return c.JSON(fiber.Map{
			"dev_eui":       req.DevEUI,
			"app_key":       req.AppKey,
			"ttn_device_id": req.TtnDeviceID,
		})
	}
}
