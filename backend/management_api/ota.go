package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
)

// GET /api/provision-config — returns WiFi credentials from env vars (JWT protected).
func getProvisionConfigHandler(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"wifi_ssid":     os.Getenv("WIFI_SSID"),
		"wifi_password": os.Getenv("WIFI_PASSWORD"),
	})
}

// POST /api/sensors/:uuid/trigger-ota (JWT protected).
// Finds the next qualifying firmware tag from Codeberg, then sends a
// CMD_OTA_UPDATE downlink containing the tag name.
func triggerOTAHandler(c fiber.Ctx) error {
	uuid := c.Params("uuid")
	if !uuidRegex.MatchString(uuid) {
		return c.Status(400).JSON(fiber.Map{"error": errInvalidSensorUUID})
	}

	// 1. Fetch sensor from backend
	resp, err := internalRequest("GET", "/internal/sensors/"+uuid, nil)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": errBackendUnreachable})
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

	currentVersion, _ := sensor["firmware_version"].(string)

	// 2. Find the next qualifying firmware tag
	channel := os.Getenv("FIRMWARE_CHANNEL")
	if channel == "" {
		channel = "stable"
	}

	tagName, err := findNextFirmwareTag(currentVersion, channel)
	if err != nil {
		if strings.HasPrefix(err.Error(), "already_latest:") {
			ver := strings.TrimPrefix(err.Error(), "already_latest:")

			return c.Status(409).JSON(fiber.Map{
				"error": fmt.Sprintf("sensor is already on the latest version (%s)", ver),
			})
		}

		return c.Status(502).JSON(fiber.Map{"error": fmt.Sprintf("failed to find next firmware tag: %v", err)})
	}

	// 3. Send downlink with CMD_OTA_UPDATE + tag name as payload
	appID, apiKey, baseURL := getTTNConfig()
	if appID == "" || apiKey == "" {
		return c.Status(503).JSON(fiber.Map{"error": "TTN not configured"})
	}

	if err := sendDownlink(baseURL, appID, apiKey, deviceID, CmdOTAUpdate, []byte(tagName)); err != nil {
		return c.Status(502).JSON(fiber.Map{"error": fmt.Sprintf("downlink failed: %v", err)})
	}

	// Determine display versions for the response message
	currentDisplay := currentVersion
	if currentDisplay == "" {
		currentDisplay = "unknown"
	}

	_, nextMinor, ok := parseTagVersion(tagName)
	nextDisplay := tagName

	if ok {
		nextMinorStr := strconv.Itoa(nextMinor)
		_ = nextMinorStr
		nextDisplay = strings.TrimPrefix(tagName, "firmware-pax-v")
	}

	return c.JSON(fiber.Map{
		"message": fmt.Sprintf("OTA queued (%s → %s) — sensor updates on next wake", currentDisplay, nextDisplay),
		"tag":     tagName,
	})
}

// findNextFirmwareTag fetches all releases from Codeberg and returns the tag name
// of the next qualifying firmware version according to channel rules:
//   - "stable": next major only (major > curMajor), minimum qualifying
//   - "develop": next minor or major ((maj,min) > (curMaj,curMin)), minimum qualifying
//
// If currentVersion is "" or unparseable, returns "firmware-pax-stable".
// Returns a sentinel error "already_latest:<version>" if no qualifying tag exists.
func findNextFirmwareTag(currentVersion, channel string) (string, error) {
	curMaj, curMin, curOk := parseVersionString(currentVersion)
	if !curOk {
		return "firmware-pax-stable", nil
	}

	// Fetch all releases from Codeberg
	url := fmt.Sprintf("%s/repos/cfhn/lora-frequenzmessung/releases?limit=50", codebergBaseURL)

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("fetch releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("codeberg releases API returned %d", resp.StatusCode)
	}

	var releases []struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return "", fmt.Errorf("parse releases: %w", err)
	}

	// Filter and sort candidates
	type candidate struct {
		tag string
		maj int
		min int
	}

	var candidates []candidate

	for _, r := range releases {
		maj, min, ok := parseTagVersion(r.TagName)
		if !ok {
			continue
		}

		switch channel {
		case "develop":
			// Any version strictly greater than current
			if maj > curMaj || (maj == curMaj && min > curMin) {
				candidates = append(candidates, candidate{r.TagName, maj, min})
			}
		default: // "stable"
			// Major version bump only
			if maj > curMaj {
				candidates = append(candidates, candidate{r.TagName, maj, min})
			}
		}
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("already_latest:%d.%d", curMaj, curMin)
	}

	// Select the minimum qualifying version
	best := candidates[0]
	for _, c := range candidates[1:] {
		if c.maj < best.maj || (c.maj == best.maj && c.min < best.min) {
			best = c
		}
	}

	return best.tag, nil
}

// parseTagVersion parses "firmware-pax-v1.3" → (1, 3, true).
func parseTagVersion(tag string) (major, minor int, ok bool) {
	s := strings.TrimPrefix(tag, "firmware-pax-v")
	if s == tag {
		return 0, 0, false
	}

	return parseVersionParts(s)
}

// parseVersionString parses a bare version like "1.2" or "1.2.0" → (1, 2, true).
func parseVersionString(v string) (major, minor int, ok bool) {
	if v == "" {
		return 0, 0, false
	}

	return parseVersionParts(v)
}

func parseVersionParts(s string) (major, minor int, ok bool) {
	parts := strings.SplitN(s, ".", 3)
	if len(parts) < 2 {
		return 0, 0, false
	}

	maj, err1 := strconv.Atoi(parts[0])

	min, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}

	return maj, min, true
}
