package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

// TestGenerateCustomsH_WithoutLoRa verifies that customs.h disables LoRa
// when no TTN credentials are provided.
func TestGenerateCustomsH_WithoutLoRa(t *testing.T) {
	result := generateCustomsH("test-uuid-1234", "", "", false)

	if !strings.Contains(result, "#define ENABLE_LORA 0") {
		t.Fatal("expected ENABLE_LORA 0 for non-TTN build")
	}
	if !strings.Contains(result, `sensor_id[] = "test-uuid-1234"`) {
		t.Fatal("expected sensor_id to be set")
	}
	if strings.Contains(result, "#define ENABLE_LORA 1") {
		t.Fatal("ENABLE_LORA must not be 1 for non-TTN build")
	}
}

// TestGenerateCustomsH_WithLoRa verifies that customs.h enables LoRa
// and includes the DevEUI/AppKey when TTN credentials are provided.
func TestGenerateCustomsH_WithLoRa(t *testing.T) {
	result := generateCustomsH("test-uuid-5678", "0011223344556677", "AABBCCDDEEFF00112233445566778899", true)

	if !strings.Contains(result, "#define ENABLE_LORA 1") {
		t.Fatal("expected ENABLE_LORA 1 for TTN build")
	}
	if !strings.Contains(result, "0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77") {
		t.Fatal("expected DevEUI bytes in output")
	}
	if !strings.Contains(result, "0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF") {
		t.Fatal("expected AppKey bytes in output")
	}
}

// TestBuildPreparation_WithoutLoRa verifies that the build directory is set up
// correctly: source is copied and customs.h is written with LoRa disabled.
func TestBuildPreparation_WithoutLoRa(t *testing.T) {
	// Create a fake sensor-pax source tree
	fakeSrc := t.TempDir()
	srcDir := filepath.Join(fakeSrc, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "main.cpp"), []byte("// main"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fakeSrc, "platformio.ini"), []byte("[env]"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("SENSOR_PAX_PATH", fakeSrc)

	_, err := runPlatformioBuild("test-uuid-no-lora", "", "", false)

	if err == nil {
		t.Fatal("expected platformio build to fail in test environment")
	}
	if strings.Contains(err.Error(), "copy source") {
		t.Fatalf("source copy should not fail: %v", err)
	}
	if strings.Contains(err.Error(), "write customs.h") {
		t.Fatalf("customs.h write should not fail: %v", err)
	}
	if strings.Contains(err.Error(), "resolve sensor-pax path") {
		t.Fatalf("path resolution should not fail: %v", err)
	}

	buildDir := filepath.Join(os.TempDir(), "pax-build-test-uuid-no-lora")

	customsPath := filepath.Join(buildDir, "src", "customs.h")
	customsBytes, err := os.ReadFile(customsPath)
	if err != nil {
		t.Fatalf("customs.h should exist in build dir: %v", err)
	}
	customs := string(customsBytes)
	if !strings.Contains(customs, "#define ENABLE_LORA 0") {
		t.Fatal("customs.h should have ENABLE_LORA 0 for non-TTN build")
	}
	if !strings.Contains(customs, `sensor_id[] = "test-uuid-no-lora"`) {
		t.Fatal("customs.h should contain the sensor UUID")
	}

	mainCpp, err := os.ReadFile(filepath.Join(buildDir, "src", "main.cpp"))
	if err != nil {
		t.Fatalf("main.cpp should be copied to build dir: %v", err)
	}
	if string(mainCpp) != "// main" {
		t.Fatal("main.cpp content should match source")
	}

	if _, err := os.Stat(filepath.Join(buildDir, "platformio.ini")); err != nil {
		t.Fatal("platformio.ini should be copied to build dir")
	}

	os.RemoveAll(buildDir)
}

// TestBuildPreparation_RelativePath verifies that a relative SENSOR_PAX_PATH
// is resolved correctly.
func TestBuildPreparation_RelativePath(t *testing.T) {
	baseDir := t.TempDir()
	fakeSrc := filepath.Join(baseDir, "sensor-pax")
	srcDir := filepath.Join(fakeSrc, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "main.cpp"), []byte("// test"), 0644); err != nil {
		t.Fatal(err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(baseDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	t.Setenv("SENSOR_PAX_PATH", "./sensor-pax")

	_, err := runPlatformioBuild("test-uuid-relpath", "", "", false)

	if err != nil && strings.Contains(err.Error(), "copy source") {
		t.Fatalf("relative path should resolve correctly: %v", err)
	}

	buildDir := filepath.Join(os.TempDir(), "pax-build-test-uuid-relpath")
	if _, err := os.Stat(filepath.Join(buildDir, "src", "customs.h")); err != nil {
		t.Fatal("build directory should be set up even with relative SENSOR_PAX_PATH")
	}

	os.RemoveAll(buildDir)
}

// TestBuildPreparation_ParentFallback verifies that when ./sensor-pax doesn't
// exist in CWD, the build falls back to ../sensor-pax.
func TestBuildPreparation_ParentFallback(t *testing.T) {
	projectRoot := t.TempDir()
	fakeSrc := filepath.Join(projectRoot, "sensor-pax", "src")
	if err := os.MkdirAll(fakeSrc, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fakeSrc, "main.cpp"), []byte("// fallback"), 0644); err != nil {
		t.Fatal(err)
	}

	backendDir := filepath.Join(projectRoot, "backend")
	if err := os.MkdirAll(backendDir, 0755); err != nil {
		t.Fatal(err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(backendDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	t.Setenv("SENSOR_PAX_PATH", "./sensor-pax")

	_, err := runPlatformioBuild("test-uuid-fallback", "", "", false)

	if err != nil && strings.Contains(err.Error(), "copy source") {
		t.Fatalf("parent directory fallback should find sensor-pax: %v", err)
	}

	buildDir := filepath.Join(os.TempDir(), "pax-build-test-uuid-fallback")
	t.Cleanup(func() { os.RemoveAll(buildDir) })

	customs, err := os.ReadFile(filepath.Join(buildDir, "src", "customs.h"))
	if err != nil {
		t.Fatalf("customs.h should exist after parent fallback: %v", err)
	}
	if !strings.Contains(string(customs), "#define ENABLE_LORA 0") {
		t.Fatal("customs.h should have ENABLE_LORA 0")
	}

	mainCpp, err := os.ReadFile(filepath.Join(buildDir, "src", "main.cpp"))
	if err != nil {
		t.Fatalf("main.cpp should be copied: %v", err)
	}
	if string(mainCpp) != "// fallback" {
		t.Fatal("main.cpp should come from ../sensor-pax, not ./sensor-pax")
	}
}

func TestGetBuildStatus_NotStarted(t *testing.T) {
	app := setupMgmtTestApp()
	buildStatuses.Delete("abcd1234")

	req := httptest.NewRequest(http.MethodGet, "/api/sensors/abcd1234/build-status", nil)
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

	req := httptest.NewRequest(http.MethodGet, "/api/sensors/abcd1234/firmware.bin", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}
