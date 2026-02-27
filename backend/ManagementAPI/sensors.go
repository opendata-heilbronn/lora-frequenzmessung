package main

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"

	"github.com/gofiber/fiber/v3"
)

func proxySensors(c fiber.Ctx) error {
	return proxyToBackend(c, "GET", "/internal/sensors", nil)
}

func proxyCreateSensor(c fiber.Ctx) error {
	return proxyToBackend(c, "POST", "/internal/sensors", bytes.NewReader(c.Body()))
}

func proxyGetSensor(c fiber.Ctx) error {
	uuid := c.Params("uuid")
	return proxyToBackend(c, "GET", "/internal/sensors/"+uuid, nil)
}

func proxyDeleteSensor(c fiber.Ctx) error {
	uuid := c.Params("uuid")

	// Fetch the sensor first to get ttn_device_id for cleanup
	resp, err := internalRequest("GET", "/internal/sensors/"+uuid, nil)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": "backend unreachable"})
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return c.Status(404).JSON(fiber.Map{"error": "sensor not found"})
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return c.Status(resp.StatusCode).Send(body)
	}

	body, _ := io.ReadAll(resp.Body)
	var sensor map[string]any
	json.Unmarshal(body, &sensor)

	// Best-effort TTN deletion before removing from DB
	if ttnDeviceID, ok := sensor["ttn_device_id"].(string); ok && ttnDeviceID != "" {
		if ttnErr := deleteTTNDevice(ttnDeviceID); ttnErr != nil {
			log.Printf("WARN: failed to delete TTN device %s: %v", ttnDeviceID, ttnErr)
		}
	}

	return proxyToBackend(c, "DELETE", "/internal/sensors/"+uuid, nil)
}

func registerTTNHandler(c fiber.Ctx) error {
	uuid := c.Params("uuid")
	if !uuidRegex.MatchString(uuid) {
		return c.Status(400).JSON(fiber.Map{"error": "invalid sensor UUID"})
	}

	// Fetch sensor to verify existence and check registration status
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

	body, _ := io.ReadAll(resp.Body)
	var sensor map[string]any
	json.Unmarshal(body, &sensor)

	if devEUI, ok := sensor["dev_eui"].(string); ok && devEUI != "" {
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

	// Persist TTN credentials to backend
	updatePayload, _ := json.Marshal(map[string]string{
		"dev_eui":       newDevEUI,
		"app_key":       appKey,
		"ttn_device_id": ttnDeviceID,
	})
	putResp, err := internalRequest("PUT", "/internal/sensors/"+uuid+"/ttn", bytes.NewReader(updatePayload))
	if err != nil || putResp.StatusCode != 200 {
		log.Printf("WARN: TTN registered but failed to persist keys for sensor %s", uuid)
		return c.Status(500).JSON(fiber.Map{"error": "TTN registered but failed to save credentials"})
	}
	putResp.Body.Close()

	return c.JSON(fiber.Map{
		"dev_eui":       newDevEUI,
		"app_key":       appKey,
		"ttn_device_id": ttnDeviceID,
	})
}

func proxyToBackend(c fiber.Ctx, method, path string, body io.Reader) error {
	resp, err := internalRequest(method, path, body)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": "backend unreachable"})
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to read backend response"})
	}

	c.Status(resp.StatusCode)
	if ct := resp.Header.Get("Content-Type"); ct != "" {
		c.Set("Content-Type", ct)
	}
	return c.Send(respBody)
}
