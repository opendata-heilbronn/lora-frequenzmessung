package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
// This tests everything in runPlatformioBuild before the platformio invocation.
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
	// Also add a platformio.ini so the copy is realistic
	if err := os.WriteFile(filepath.Join(fakeSrc, "platformio.ini"), []byte("[env]"), 0644); err != nil {
		t.Fatal(err)
	}

	// Point SENSOR_PAX_PATH to our fake source
	t.Setenv("SENSOR_PAX_PATH", fakeSrc)

	// Run the build — it will fail at the platformio step, which is expected
	_, err := runPlatformioBuild("test-uuid-no-lora", "", "", false)

	// We expect a platformio error (not installed or not a real project),
	// but NOT a "copy source" or "write customs.h" error
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

	// Verify the build directory was created with correct contents
	buildDir := filepath.Join(os.TempDir(), "pax-build-test-uuid-no-lora")

	// customs.h should exist and have ENABLE_LORA 0
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

	// main.cpp should have been copied
	mainCpp, err := os.ReadFile(filepath.Join(buildDir, "src", "main.cpp"))
	if err != nil {
		t.Fatalf("main.cpp should be copied to build dir: %v", err)
	}
	if string(mainCpp) != "// main" {
		t.Fatal("main.cpp content should match source")
	}

	// platformio.ini should be present
	if _, err := os.Stat(filepath.Join(buildDir, "platformio.ini")); err != nil {
		t.Fatal("platformio.ini should be copied to build dir")
	}

	// Cleanup
	os.RemoveAll(buildDir)
}

// TestBuildPreparation_RelativePath verifies that a relative SENSOR_PAX_PATH
// is resolved correctly (the fix for the ./sensor-pax lstat error).
func TestBuildPreparation_RelativePath(t *testing.T) {
	// Create a fake sensor-pax in a subdirectory
	baseDir := t.TempDir()
	fakeSrc := filepath.Join(baseDir, "sensor-pax")
	srcDir := filepath.Join(fakeSrc, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "main.cpp"), []byte("// test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Change to baseDir so "./sensor-pax" resolves correctly
	origDir, _ := os.Getwd()
	if err := os.Chdir(baseDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	t.Setenv("SENSOR_PAX_PATH", "./sensor-pax")

	_, err := runPlatformioBuild("test-uuid-relpath", "", "", false)

	// Should fail at platformio, NOT at source copy
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
// exist in CWD, the build falls back to ../sensor-pax (the real-world scenario
// where the backend runs from backend/ but sensor-pax is at the project root).
func TestBuildPreparation_ParentFallback(t *testing.T) {
	// Create project-root/sensor-pax/src/
	projectRoot := t.TempDir()
	fakeSrc := filepath.Join(projectRoot, "sensor-pax", "src")
	if err := os.MkdirAll(fakeSrc, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fakeSrc, "main.cpp"), []byte("// fallback"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create project-root/backend/ (simulate backend CWD — no sensor-pax here)
	backendDir := filepath.Join(projectRoot, "backend")
	if err := os.MkdirAll(backendDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Change to backend/ so ./sensor-pax doesn't exist, but ../sensor-pax does
	origDir, _ := os.Getwd()
	if err := os.Chdir(backendDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	t.Setenv("SENSOR_PAX_PATH", "./sensor-pax")

	_, err := runPlatformioBuild("test-uuid-fallback", "", "", false)

	// Must NOT fail with "copy source" — the parent fallback should kick in
	if err != nil && strings.Contains(err.Error(), "copy source") {
		t.Fatalf("parent directory fallback should find sensor-pax: %v", err)
	}

	buildDir := filepath.Join(os.TempDir(), "pax-build-test-uuid-fallback")
	t.Cleanup(func() { os.RemoveAll(buildDir) })

	// Verify customs.h was written
	customs, err := os.ReadFile(filepath.Join(buildDir, "src", "customs.h"))
	if err != nil {
		t.Fatalf("customs.h should exist after parent fallback: %v", err)
	}
	if !strings.Contains(string(customs), "#define ENABLE_LORA 0") {
		t.Fatal("customs.h should have ENABLE_LORA 0")
	}

	// Verify source was copied from the parent's sensor-pax
	mainCpp, err := os.ReadFile(filepath.Join(buildDir, "src", "main.cpp"))
	if err != nil {
		t.Fatalf("main.cpp should be copied: %v", err)
	}
	if string(mainCpp) != "// fallback" {
		t.Fatal("main.cpp should come from ../sensor-pax, not ./sensor-pax")
	}
}
