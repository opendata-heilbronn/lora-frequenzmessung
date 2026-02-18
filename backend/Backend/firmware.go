package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5"
)

type buildState struct {
	Status  string `json:"status"` // "building" | "done" | "error"
	Message string `json:"message,omitempty"`
	BinPath string `json:"-"`
}

var buildStatuses sync.Map // key: uuid → *buildState

func buildFirmware(conn *pgx.Conn, dbCtx context.Context) fiber.Handler {
	return func(c fiber.Ctx) error {
		uuid := c.Params("uuid")

		var devEUI, appKey string
		err := conn.QueryRow(dbCtx,
			`SELECT COALESCE(dev_eui,''), COALESCE(app_key,'') FROM sensors WHERE uuid = $1`, uuid).
			Scan(&devEUI, &appKey)
		if err == pgx.ErrNoRows {
			return c.Status(404).JSON(fiber.Map{"error": "sensor not found"})
		}
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		// Check if already building
		if st, ok := buildStatuses.Load(uuid); ok {
			if st.(*buildState).Status == "building" {
				return c.Status(202).JSON(fiber.Map{"status": "already building"})
			}
		}

		state := &buildState{Status: "building"}
		buildStatuses.Store(uuid, state)

		hasTTN := devEUI != ""

		go func() {
			binPath, err := runPlatformioBuild(uuid, devEUI, appKey, hasTTN)
			if err != nil {
				state.Status = "error"
				state.Message = err.Error()
				return
			}
			state.Status = "done"
			state.BinPath = binPath
		}()

		return c.Status(202).JSON(fiber.Map{"status": "building"})
	}
}

func getBuildStatus() fiber.Handler {
	return func(c fiber.Ctx) error {
		uuid := c.Params("uuid")
		st, ok := buildStatuses.Load(uuid)
		if !ok {
			return c.Status(404).JSON(fiber.Map{"status": "not_started"})
		}
		return c.JSON(st.(*buildState))
	}
}

func getFirmwareManifest(conn *pgx.Conn, dbCtx context.Context) fiber.Handler {
	return func(c fiber.Ctx) error {
		uuid := c.Params("uuid")

		var name string
		err := conn.QueryRow(dbCtx, `SELECT name FROM sensors WHERE uuid = $1`, uuid).Scan(&name)
		if err == pgx.ErrNoRows {
			return c.Status(404).JSON(fiber.Map{"error": "sensor not found"})
		}
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		manifest := map[string]any{
			"name":    name,
			"version": "1.0.0",
			"builds": []map[string]any{
				{
					"chipFamily": "ESP32-S3",
					"parts": []map[string]any{
						{"path": fmt.Sprintf("/api/sensors/%s/firmware.bin", uuid), "offset": 65536},
					},
				},
			},
		}

		c.Set("Content-Type", "application/json")
		return c.JSON(manifest)
	}
}

func getFirmwareBin() fiber.Handler {
	return func(c fiber.Ctx) error {
		uuid := c.Params("uuid")

		st, ok := buildStatuses.Load(uuid)
		if !ok || st.(*buildState).Status != "done" {
			return c.Status(404).JSON(fiber.Map{"error": "firmware not built yet"})
		}

		binPath := st.(*buildState).BinPath
		if _, err := os.Stat(binPath); os.IsNotExist(err) {
			return c.Status(404).JSON(fiber.Map{"error": "firmware binary not found"})
		}

		c.Set("Content-Type", "application/octet-stream")
		return c.SendFile(binPath)
	}
}

func runPlatformioBuild(uuid, devEUI, appKey string, hasTTN bool) (string, error) {
	// Determine source sensor-pax path
	sensorPaxSrc := os.Getenv("SENSOR_PAX_PATH")
	if sensorPaxSrc == "" {
		sensorPaxSrc = "./sensor-pax"
	}
	// Resolve to absolute path
	sensorPaxSrc, err := filepath.Abs(sensorPaxSrc)
	if err != nil {
		return "", fmt.Errorf("resolve sensor-pax path: %w", err)
	}
	// If the resolved path doesn't exist, try ../sensor-pax (project root when
	// the backend runs from backend/).
	if _, err := os.Stat(sensorPaxSrc); os.IsNotExist(err) {
		alt, absErr := filepath.Abs(filepath.Join("..", "sensor-pax"))
		if absErr == nil {
			if _, statErr := os.Stat(alt); statErr == nil {
				sensorPaxSrc = alt
			}
		}
	}

	// Create temp build directory
	buildDir := filepath.Join(os.TempDir(), fmt.Sprintf("pax-build-%s", uuid))
	if err := os.RemoveAll(buildDir); err != nil {
		return "", fmt.Errorf("cleanup build dir: %w", err)
	}

	// Copy sensor-pax to build dir
	if err := copyDir(sensorPaxSrc, buildDir); err != nil {
		return "", fmt.Errorf("copy source: %w", err)
	}

	// Generate customs.h
	customsH := generateCustomsH(uuid, devEUI, appKey, hasTTN)
	customsPath := filepath.Join(buildDir, "src", "customs.h")
	if err := os.WriteFile(customsPath, []byte(customsH), 0644); err != nil {
		return "", fmt.Errorf("write customs.h: %w", err)
	}

	// Run platformio build
	cmd := exec.Command("platformio", "run")
	cmd.Dir = buildDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("platformio build failed: %w\n%s", err, string(out))
	}

	// Copy firmware.bin to a served location
	firmwareSrc := filepath.Join(buildDir, ".pio", "build", "heltec_wifi_lora_32_V3", "firmware.bin")
	firmwareDst := filepath.Join(os.TempDir(), "firmware", uuid, "firmware.bin")
	if err := os.MkdirAll(filepath.Dir(firmwareDst), 0755); err != nil {
		return "", fmt.Errorf("create firmware dir: %w", err)
	}
	if err := copyFile(firmwareSrc, firmwareDst); err != nil {
		return "", fmt.Errorf("copy firmware.bin: %w", err)
	}

	return firmwareDst, nil
}

func generateCustomsH(sensorID, devEUI, appKey string, hasTTN bool) string {
	enableLora := 0
	devEuiArr := "0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00"
	appKeyArr := "0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00"
	if hasTTN {
		enableLora = 1
		devEuiArr = hexToCArray(devEUI)
		appKeyArr = hexToCArray(appKey)
	}

	return fmt.Sprintf(`#ifndef CUSTOMS_H
#define CUSTOMS_H

#include <cstdint>

#define ENABLE_LOGGING 1
#define ENABLE_DISPLAY 0
#define ENABLE_LORA %d

char sensor_id[] = "%s";

uint8_t devEui[] = {%s};
uint8_t appEui[] = {0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01};
uint8_t appKey[] = {%s};

/* ABP para (unused for OTAA) */
uint8_t nwkSKey[] = {0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00};
uint8_t appSKey[] = {0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00};
uint32_t devAddr = (uint32_t)0x00000000;
uint16_t userChannelsMask[6] = {0x00FF, 0x0000, 0x0000, 0x0000, 0x0000, 0x0000};

float factor = 0.7;
float sleepTime = 900;

int SensorTypFrequency = 0;
int SensorTypBattery = 1;

#endif // CUSTOMS_H
`,
		enableLora,
		sensorID,
		devEuiArr,
		appKeyArr,
	)
}

// hexToCArray converts an uppercase hex string to a C array initializer.
// e.g. "70B3D57E" → "0x70, 0xB3, 0xD5, 0x7E"
func hexToCArray(hexStr string) string {
	hexStr = strings.ToUpper(hexStr)
	parts := make([]string, 0, len(hexStr)/2)
	for i := 0; i+1 < len(hexStr); i += 2 {
		parts = append(parts, "0x"+hexStr[i:i+2])
	}
	return strings.Join(parts, ", ")
}

// manifestJSON returns the esp-web-tools manifest as a JSON string (used internally).
func manifestJSON(name, uuid string) string {
	m := map[string]any{
		"name":    name,
		"version": "1.0.0",
		"builds": []map[string]any{
			{
				"chipFamily": "ESP32-S3",
				"parts": []map[string]any{
					{"path": fmt.Sprintf("/api/sensors/%s/firmware.bin", uuid), "offset": 65536},
				},
			},
		},
	}
	b, _ := json.Marshal(m)
	return string(b)
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Skip .pio build artifacts to keep build clean
		if info.IsDir() && info.Name() == ".pio" {
			return filepath.SkipDir
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}
		return copyFile(path, dstPath)
	})
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}
