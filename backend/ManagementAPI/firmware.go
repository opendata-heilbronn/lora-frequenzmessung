package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
)

type buildState struct {
	mu      sync.Mutex
	Status  string `json:"status"` // "building" | "done" | "error"
	Message string `json:"message,omitempty"`
	BinPath string `json:"-"` // path to firmware.bin in output dir
}

var buildStatuses sync.Map // key: uuid → *buildState

// buildSemaphore limits concurrent firmware builds.
var buildSemaphore = make(chan struct{}, 2)

const buildTimeout = 5 * time.Minute

func buildFirmwareHandler(c fiber.Ctx) error {
	uuid := c.Params("uuid")
	if !uuidRegex.MatchString(uuid) {
		return c.Status(400).JSON(fiber.Map{"error": "invalid sensor UUID"})
	}

	// Fetch sensor from backend to get TTN credentials
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

	devEUI, _ := sensor["dev_eui"].(string)
	appKey, _ := sensor["app_key"].(string)

	// Check if already building
	if st, ok := buildStatuses.Load(uuid); ok {
		bs := st.(*buildState)
		bs.mu.Lock()
		status := bs.Status
		bs.mu.Unlock()
		if status == "building" {
			return c.Status(202).JSON(fiber.Map{"status": "already building"})
		}
	}

	state := &buildState{Status: "building"}
	buildStatuses.Store(uuid, state)

	hasTTN := devEUI != ""

	go func() {
		// Acquire build semaphore
		buildSemaphore <- struct{}{}
		defer func() { <-buildSemaphore }()

		binPath, err := runPlatformioBuild(uuid, devEUI, appKey, hasTTN)
		state.mu.Lock()
		defer state.mu.Unlock()
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

func getBuildStatusHandler(c fiber.Ctx) error {
	uuid := c.Params("uuid")
	if !uuidRegex.MatchString(uuid) {
		return c.Status(400).JSON(fiber.Map{"error": "invalid sensor UUID"})
	}
	st, ok := buildStatuses.Load(uuid)
	if !ok {
		return c.Status(404).JSON(fiber.Map{"status": "not_started"})
	}
	bs := st.(*buildState)
	bs.mu.Lock()
	defer bs.mu.Unlock()
	return c.JSON(fiber.Map{
		"status":  bs.Status,
		"message": bs.Message,
	})
}

func getManifestHandler(c fiber.Ctx) error {
	uuid := c.Params("uuid")
	if !uuidRegex.MatchString(uuid) {
		return c.Status(400).JSON(fiber.Map{"error": "invalid sensor UUID"})
	}

	// Fetch sensor name from backend
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
	name, _ := sensor["name"].(string)

	// Manifest used by esp-web-tools. For ESP32-S3 we flash 3 parts like the CLI example:
	//   0x0      bootloader.bin
	//   0x8000   partitions.bin
	//   0x10000  firmware.bin
	parts := []map[string]any{
		{"path": fmt.Sprintf("/api/sensors/%s/bootloader.bin", uuid), "offset": 0x0},    // 0
		{"path": fmt.Sprintf("/api/sensors/%s/partitions.bin", uuid), "offset": 0x8000}, // 32768
		{"path": fmt.Sprintf("/api/sensors/%s/firmware.bin", uuid), "offset": 0x10000},  // 65536
	}

	manifest := map[string]any{
		"name":    name,
		"version": "1.1.0",
		"builds": []map[string]any{
			{
				"chipFamily": "ESP32-S3",
				"parts":      parts,
			},
		},
	}

	c.Set("Content-Type", "application/json")
	c.Set("Cache-Control", "no-store")
	return c.JSON(manifest)
}

func getFirmwareBinHandler(c fiber.Ctx) error {
	uuid := c.Params("uuid")
	if !uuidRegex.MatchString(uuid) {
		return c.Status(400).JSON(fiber.Map{"error": "invalid sensor UUID"})
	}

	st, ok := buildStatuses.Load(uuid)
	if !ok {
		return c.Status(404).JSON(fiber.Map{"error": "firmware not built yet"})
	}
	bs := st.(*buildState)
	bs.mu.Lock()
	status := bs.Status
	bs.mu.Unlock()

	if status != "done" {
		return c.Status(404).JSON(fiber.Map{"error": "firmware not built yet"})
	}

	binPath := filepath.Join(os.TempDir(), "firmware", uuid, "firmware.bin")
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		return c.Status(404).JSON(fiber.Map{"error": "firmware binary not found"})
	}

	c.Set("Content-Type", "application/octet-stream")
	c.Set("Cache-Control", "no-store")
	return c.SendFile(binPath)
}

func getBootloaderBinHandler(c fiber.Ctx) error {
	uuid := c.Params("uuid")
	if !uuidRegex.MatchString(uuid) {
		return c.Status(400).JSON(fiber.Map{"error": "invalid sensor UUID"})
	}
	st, ok := buildStatuses.Load(uuid)
	if !ok {
		return c.Status(404).JSON(fiber.Map{"error": "firmware not built yet"})
	}
	bs := st.(*buildState)
	bs.mu.Lock()
	status := bs.Status
	bs.mu.Unlock()
	if status != "done" {
		return c.Status(404).JSON(fiber.Map{"error": "firmware not built yet"})
	}
	p := filepath.Join(os.TempDir(), "firmware", uuid, "bootloader.bin")
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return c.Status(404).JSON(fiber.Map{"error": "bootloader.bin not found"})
	}
	c.Set("Content-Type", "application/octet-stream")
	c.Set("Cache-Control", "no-store")
	return c.SendFile(p)
}

func getPartitionsBinHandler(c fiber.Ctx) error {
	uuid := c.Params("uuid")
	if !uuidRegex.MatchString(uuid) {
		return c.Status(400).JSON(fiber.Map{"error": "invalid sensor UUID"})
	}
	st, ok := buildStatuses.Load(uuid)
	if !ok {
		return c.Status(404).JSON(fiber.Map{"error": "firmware not built yet"})
	}
	bs := st.(*buildState)
	bs.mu.Lock()
	status := bs.Status
	bs.mu.Unlock()
	if status != "done" {
		return c.Status(404).JSON(fiber.Map{"error": "firmware not built yet"})
	}
	p := filepath.Join(os.TempDir(), "firmware", uuid, "partitions.bin")
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return c.Status(404).JSON(fiber.Map{"error": "partitions.bin not found"})
	}
	c.Set("Content-Type", "application/octet-stream")
	c.Set("Cache-Control", "no-store")
	return c.SendFile(p)
}

func getOtaDataBinHandler(c fiber.Ctx) error {
	uuid := c.Params("uuid")
	if !uuidRegex.MatchString(uuid) {
		return c.Status(400).JSON(fiber.Map{"error": "invalid sensor UUID"})
	}
	st, ok := buildStatuses.Load(uuid)
	if !ok {
		return c.Status(404).JSON(fiber.Map{"error": "firmware not built yet"})
	}
	bs := st.(*buildState)
	bs.mu.Lock()
	status := bs.Status
	bs.mu.Unlock()
	if status != "done" {
		return c.Status(404).JSON(fiber.Map{"error": "firmware not built yet"})
	}
	p := filepath.Join(os.TempDir(), "firmware", uuid, "otadata.bin")
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return c.Status(404).JSON(fiber.Map{"error": "otadata.bin not found"})
	}
	c.Set("Content-Type", "application/octet-stream")
	c.Set("Cache-Control", "no-store")
	return c.SendFile(p)
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

	// Generate customs.h (restrictive permissions — contains keys)
	customsH := generateCustomsH(uuid, devEUI, appKey, hasTTN)
	customsPath := filepath.Join(buildDir, "src", "customs.h")
	if err := os.WriteFile(customsPath, []byte(customsH), 0600); err != nil {
		return "", fmt.Errorf("write customs.h: %w", err)
	}

	// Run platformio build with timeout
	ctx, cancel := context.WithTimeout(context.Background(), buildTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "platformio", "run")
	cmd.Dir = buildDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("platformio build failed: %w\n%s", err, string(out))
	}

	// Locate build artifacts
	outDir := filepath.Join(buildDir, ".pio", "build", "heltec_wifi_lora_32_V3")
	bootloaderSrc := filepath.Join(outDir, "bootloader.bin")
	partitionsSrc := filepath.Join(outDir, "partitions.bin")
	// Espressif sometimes emits ota_data_initial.bin or boot_app0.bin
	otaInitSrc := filepath.Join(outDir, "ota_data_initial.bin")
	bootApp0Src := filepath.Join(outDir, "boot_app0.bin")
	firmwareSrc := filepath.Join(outDir, "firmware.bin")

	// Destination directory for serving
	dstDir := filepath.Join(os.TempDir(), "firmware", uuid)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return "", fmt.Errorf("create firmware dir: %w", err)
	}

	// Copy required artifacts
	if err := copyFile(bootloaderSrc, filepath.Join(dstDir, "bootloader.bin")); err != nil {
		return "", fmt.Errorf("copy bootloader.bin: %w", err)
	}
	if err := copyFile(partitionsSrc, filepath.Join(dstDir, "partitions.bin")); err != nil {
		return "", fmt.Errorf("copy partitions.bin: %w", err)
	}
	// otadata: prefer ota_data_initial.bin; fallback to boot_app0.bin; if neither exists, synthesize 0x2000 bytes of 0xFF
	otaDst := filepath.Join(dstDir, "otadata.bin")
	if _, err := os.Stat(otaInitSrc); err == nil {
		if err := copyFile(otaInitSrc, otaDst); err != nil {
			return "", fmt.Errorf("copy ota_data_initial.bin: %w", err)
		}
	} else if _, err := os.Stat(bootApp0Src); err == nil {
		if err := copyFile(bootApp0Src, otaDst); err != nil {
			return "", fmt.Errorf("copy boot_app0.bin as otadata.bin: %w", err)
		}
	} else {
		// Synthesize a valid (erased) OTA data sector: 0x2000 (8192) bytes of 0xFF
		f, err := os.Create(otaDst)
		if err != nil {
			return "", fmt.Errorf("create synthesized otadata.bin: %w", err)
		}
		defer f.Close()
		buf := bytes.Repeat([]byte{0xFF}, 0x2000)
		if _, err := f.Write(buf); err != nil {
			return "", fmt.Errorf("write synthesized otadata.bin: %w", err)
		}
		if err := f.Sync(); err != nil {
			return "", fmt.Errorf("sync synthesized otadata.bin: %w", err)
		}
		log.Printf("INFO: synthesized otadata.bin (8192 bytes of 0xFF) for sensor %s", uuid)
	}
	if err := copyFile(firmwareSrc, filepath.Join(dstDir, "firmware.bin")); err != nil {
		return "", fmt.Errorf("copy firmware.bin: %w", err)
	}

	// Clean up build directory now that firmware binaries have been copied
	if err := os.RemoveAll(buildDir); err != nil {
		log.Printf("WARN: failed to clean up build dir %s: %v", buildDir, err)
	}

	return filepath.Join(dstDir, "firmware.bin"), nil
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
