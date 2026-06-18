package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/gofiber/fiber/v3"
)

// StartVersionCheckScheduler starts a background goroutine that sends a
// CmdRequestVersion downlink to all TTN-linked sensors on every tick.
func StartVersionCheckScheduler(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			if err := sendVersionRequestToAllSensors(); err != nil {
				log.Printf("WARN: scheduled version check failed: %v", err)
			}
		}
	}()
}

func sendVersionRequestToAllSensors() error {
	resp, err := internalRequest("GET", "/internal/sensors", nil)
	if err != nil {
		return fmt.Errorf("fetch sensors: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read sensors response: %w", err)
	}

	var sensors []map[string]any
	if err := json.Unmarshal(body, &sensors); err != nil {
		return fmt.Errorf("parse sensors: %w", err)
	}

	appID, apiKey, baseURL := getTTNConfig()
	if appID == "" || apiKey == "" {
		return fmt.Errorf("TTN not configured")
	}

	for _, s := range sensors {
		deviceID, _ := s["ttn_device_id"].(string)
		if deviceID == "" {
			continue
		}

		if err := sendDownlink(baseURL, appID, apiKey, deviceID, CmdRequestVersion, nil); err != nil {
			log.Printf("WARN: version downlink to %s failed: %v", deviceID, err)
		}
	}

	return nil
}

// requestVersionHandler sends a CmdRequestVersion downlink to a single sensor.
func requestVersionHandler(c fiber.Ctx) error {
	uuid := c.Params("uuid")
	if !uuidRegex.MatchString(uuid) {
		return c.Status(400).JSON(fiber.Map{"error": "invalid sensor UUID"})
	}

	resp, err := internalRequest("GET", "/internal/sensors/"+uuid, nil)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": "backend unreachable"})
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return c.Status(404).JSON(fiber.Map{"error": "sensor not found"})
	}

	if resp.StatusCode != 200 {
		return c.Status(resp.StatusCode).JSON(fiber.Map{"error": "backend error"})
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to read sensor"})
	}

	var sensor map[string]any
	if err := json.Unmarshal(body, &sensor); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "invalid sensor response"})
	}

	deviceID, _ := sensor["ttn_device_id"].(string)
	if deviceID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "sensor is not linked to TTN"})
	}

	appID, apiKey, baseURL := getTTNConfig()
	if appID == "" || apiKey == "" {
		return c.Status(503).JSON(fiber.Map{"error": "TTN not configured"})
	}

	if err := sendDownlink(baseURL, appID, apiKey, deviceID, CmdRequestVersion, nil); err != nil {
		return c.Status(502).JSON(fiber.Map{"error": fmt.Sprintf("downlink failed: %v", err)})
	}

	return c.JSON(fiber.Map{"message": "version request queued — sensor will report on next wake"})
}

// requestAllVersionsHandler sends CmdRequestVersion to all TTN-linked sensors.
func requestAllVersionsHandler(c fiber.Ctx) error {
	if err := sendVersionRequestToAllSensors(); err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "version request queued for all linked sensors"})
}
