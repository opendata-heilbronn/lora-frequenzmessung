package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v3"
)

// setupMgmtTestApp creates a Fiber app with firmware routes for testing.
func setupMgmtTestApp() *fiber.App {
	app := fiber.New()
	app.Get("/api/sensors/:uuid/build-status", getBuildStatusHandler)
	app.Get("/api/sensors/:uuid/firmware.bin", getFirmwareBinHandler)

	return app
}

// mockCodebergServer returns an httptest.Server that simulates the Codeberg release API.
// assets is a map of filename → content.
func mockCodebergServer(t *testing.T, tag string, assets map[string][]byte) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	// Serve release metadata
	mux.HandleFunc("/api/v1/repos/cfhn/lora-frequenzmessung/releases/tags/"+tag, func(w http.ResponseWriter, r *http.Request) {
		type asset struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		}

		var assetList []asset
		for name := range assets {
			assetList = append(assetList, asset{
				Name:               name,
				BrowserDownloadURL: "http://" + r.Host + "/download/" + name,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"tag_name": tag, "assets": assetList})
	})

	// Serve binary downloads
	mux.HandleFunc("/download/", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Path[len("/download/"):]

		content, ok := assets[name]
		if !ok {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(content)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return srv
}

// TestDownloadFirmware_Stable verifies that FIRMWARE_CHANNEL=stable downloads
// the three required binaries and synthesizes otadata.bin.
func TestDownloadFirmware_Stable(t *testing.T) {
	assets := map[string][]byte{
		"heltec_wifi_lora_32_V3_bootloader.bin": []byte("bootloader-data"),
		"heltec_wifi_lora_32_V3_partitions.bin": []byte("partitions-data"),
		"heltec_wifi_lora_32_V3_firmware.bin":   []byte("firmware-data"),
	}
	srv := mockCodebergServer(t, "firmware-pax-stable", assets)

	orig := codebergBaseURL
	codebergBaseURL = srv.URL + "/api/v1"

	t.Cleanup(func() { codebergBaseURL = orig })

	t.Setenv("FIRMWARE_CHANNEL", "stable")

	uuid := "aabbccdd"
	dstDir := filepath.Join(os.TempDir(), "firmware", uuid)

	t.Cleanup(func() { os.RemoveAll(dstDir) })

	binPath, tagName, err := downloadFirmwareRelease(uuid)
	if err != nil {
		t.Fatalf("downloadFirmwareRelease failed: %v", err)
	}

	if tagName != "firmware-pax-stable" {
		t.Errorf("unexpected tagName: %s", tagName)
	}

	if binPath != filepath.Join(dstDir, "firmware.bin") {
		t.Errorf("unexpected binPath: %s", binPath)
	}

	// Verify the 3 downloaded files
	for _, name := range []string{"bootloader.bin", "partitions.bin", "firmware.bin"} {
		data, err := os.ReadFile(filepath.Join(dstDir, name))
		if err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}

		if len(data) == 0 {
			t.Errorf("%s is empty", name)
		}
	}

	// Verify otadata.bin is synthesized: 8192 bytes of 0xFF
	otadata, err := os.ReadFile(filepath.Join(dstDir, "otadata.bin"))
	if err != nil {
		t.Fatalf("missing otadata.bin: %v", err)
	}

	if len(otadata) != 0x2000 {
		t.Errorf("otadata.bin: expected 8192 bytes, got %d", len(otadata))
	}

	for i, b := range otadata {
		if b != 0xFF {
			t.Errorf("otadata.bin[%d] = 0x%02X, want 0xFF", i, b)
			break
		}
	}
}

// TestDownloadFirmware_DevelopChannel verifies that FIRMWARE_CHANNEL=develop
// uses the firmware-pax-develop tag.
func TestDownloadFirmware_DevelopChannel(t *testing.T) {
	assets := map[string][]byte{
		"heltec_wifi_lora_32_V3_bootloader.bin": []byte("bl"),
		"heltec_wifi_lora_32_V3_partitions.bin": []byte("pt"),
		"heltec_wifi_lora_32_V3_firmware.bin":   []byte("fw"),
	}
	srv := mockCodebergServer(t, "firmware-pax-develop", assets)

	orig := codebergBaseURL
	codebergBaseURL = srv.URL + "/api/v1"

	t.Cleanup(func() { codebergBaseURL = orig })

	t.Setenv("FIRMWARE_CHANNEL", "develop")

	uuid := "11223344"
	dstDir := filepath.Join(os.TempDir(), "firmware", uuid)

	t.Cleanup(func() { os.RemoveAll(dstDir) })

	_, _, err := downloadFirmwareRelease(uuid)
	if err != nil {
		t.Fatalf("downloadFirmwareRelease (develop channel) failed: %v", err)
	}
}

// TestDownloadFirmware_MissingAsset verifies that a missing asset returns an error.
func TestDownloadFirmware_MissingAsset(t *testing.T) {
	// Only provide bootloader and partitions — firmware.bin is missing
	assets := map[string][]byte{
		"heltec_wifi_lora_32_V3_bootloader.bin": []byte("bl"),
		"heltec_wifi_lora_32_V3_partitions.bin": []byte("pt"),
	}
	srv := mockCodebergServer(t, "firmware-pax-stable", assets)

	orig := codebergBaseURL
	codebergBaseURL = srv.URL + "/api/v1"

	t.Cleanup(func() { codebergBaseURL = orig })

	t.Setenv("FIRMWARE_CHANNEL", "stable")

	uuid := "deadbeef"
	dstDir := filepath.Join(os.TempDir(), "firmware", uuid)

	t.Cleanup(func() { os.RemoveAll(dstDir) })

	_, _, err := downloadFirmwareRelease(uuid)
	if err == nil {
		t.Fatal("expected error for missing firmware.bin asset, got nil")
	}
}

func TestGetBuildStatus_NotStarted(t *testing.T) {
	app := setupMgmtTestApp()

	buildStatuses.Delete("abcd1234")

	req := httptest.NewRequest(http.MethodGet, "/api/sensors/abcd1234/build-status", http.NoBody)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 404 {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestGetFirmwareBin_NotBuilt(t *testing.T) {
	app := setupMgmtTestApp()

	buildStatuses.Delete("abcd1234")

	req := httptest.NewRequest(http.MethodGet, "/api/sensors/abcd1234/firmware.bin", http.NoBody)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 404 {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}
