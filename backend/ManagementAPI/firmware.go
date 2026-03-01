package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/gofiber/fiber/v3"
)

type buildState struct {
	mu      sync.Mutex
	Status  string `json:"status"` // "building" | "done" | "error"
	Message string `json:"message,omitempty"`
	BinPath string `json:"-"` // path to firmware.bin in output dir
}

var buildStatuses sync.Map // key: uuid → *buildState

func buildFirmwareHandler(c fiber.Ctx) error {
	uuid := c.Params("uuid")
	if !uuidRegex.MatchString(uuid) {
		return c.Status(400).JSON(fiber.Map{"error": "invalid sensor UUID"})
	}

	// Verify sensor exists in backend
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

	go func() {
		binPath, err := downloadFirmwareRelease(uuid)
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

	// Manifest used by esp-web-tools. For ESP32-S3 we flash 3 parts:
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

// codebergBaseURL is the base URL for Codeberg API calls. Can be overridden in tests.
var codebergBaseURL = "https://codeberg.org/api/v1"

// downloadFirmwareRelease downloads pre-built firmware binaries from a Codeberg release
// into /tmp/firmware/{uuid}/ and synthesizes otadata.bin.
func downloadFirmwareRelease(uuid string) (string, error) {
	channel := os.Getenv("FIRMWARE_CHANNEL")
	if channel == "" {
		channel = "stable"
	}

	tag := "firmware-pax-stable"
	if channel == "develop" {
		tag = "firmware-pax-develop"
	}

	// Fetch release metadata from Codeberg API
	releaseURL := fmt.Sprintf("%s/repos/cfhn/lora-frequenzmessung/releases/tags/%s", codebergBaseURL, tag)
	releaseResp, err := http.Get(releaseURL)
	if err != nil {
		return "", fmt.Errorf("fetch release metadata: %w", err)
	}
	defer releaseResp.Body.Close()

	if releaseResp.StatusCode != 200 {
		return "", fmt.Errorf("codeberg release API returned %d for tag %s", releaseResp.StatusCode, tag)
	}

	var release struct {
		Assets []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(releaseResp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("parse release metadata: %w", err)
	}

	// Helper to find an asset by suffix
	findAsset := func(suffix string) (string, error) {
		for _, a := range release.Assets {
			if len(a.Name) >= len(suffix) && a.Name[len(a.Name)-len(suffix):] == suffix {
				return a.BrowserDownloadURL, nil
			}
		}
		return "", fmt.Errorf("asset with suffix %q not found in release %s", suffix, tag)
	}

	bootloaderURL, err := findAsset("_bootloader.bin")
	if err != nil {
		return "", err
	}
	partitionsURL, err := findAsset("_partitions.bin")
	if err != nil {
		return "", err
	}
	firmwareURL, err := findAsset("_firmware.bin")
	if err != nil {
		return "", err
	}

	// Prepare destination directory
	dstDir := filepath.Join(os.TempDir(), "firmware", uuid)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return "", fmt.Errorf("create firmware dir: %w", err)
	}

	// Download each binary
	type download struct {
		url  string
		name string
	}
	downloads := []download{
		{bootloaderURL, "bootloader.bin"},
		{partitionsURL, "partitions.bin"},
		{firmwareURL, "firmware.bin"},
	}

	for _, d := range downloads {
		if err := downloadFile(d.url, filepath.Join(dstDir, d.name)); err != nil {
			return "", fmt.Errorf("download %s: %w", d.name, err)
		}
	}

	// Synthesize otadata.bin: 8192 bytes of 0xFF (erased OTA data sector)
	otaDst := filepath.Join(dstDir, "otadata.bin")
	f, err := os.Create(otaDst)
	if err != nil {
		return "", fmt.Errorf("create otadata.bin: %w", err)
	}
	buf := make([]byte, 0x2000)
	for i := range buf {
		buf[i] = 0xFF
	}
	if _, err := f.Write(buf); err != nil {
		f.Close()
		return "", fmt.Errorf("write otadata.bin: %w", err)
	}
	if err := f.Close(); err != nil {
		return "", fmt.Errorf("close otadata.bin: %w", err)
	}
	log.Printf("INFO: synthesized otadata.bin (8192 bytes of 0xFF) for sensor %s", uuid)

	return filepath.Join(dstDir, "firmware.bin"), nil
}

// downloadFile downloads a URL and writes it to dst.
func downloadFile(url, dst string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}
