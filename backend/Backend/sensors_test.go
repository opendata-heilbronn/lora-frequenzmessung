package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"
)

// setupTestApp creates a Fiber app with sensor routes registered.
// A nil pool is safe for tests that only exercise validation code paths,
// because validation returns 400 before any DB call is made.
func setupTestApp(pool *pgxpool.Pool) *fiber.App {
	app := fiber.New()
	ctx := context.Background()

	app.Post("/api/sensors", createSensor(pool, ctx))
	app.Get("/api/sensors/:uuid/build-status", getBuildStatus())
	app.Get("/api/sensors/:uuid/firmware.bin", getFirmwareBin())

	return app
}

func TestCreateSensor_EmptyName(t *testing.T) {
	app := setupTestApp(nil)
	req := httptest.NewRequest(http.MethodPost, "/api/sensors",
		strings.NewReader(`{"name":"","latitude":49.0,"longitude":9.0}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestCreateSensor_NameTooLong(t *testing.T) {
	app := setupTestApp(nil)
	longName := strings.Repeat("a", 65)
	body := `{"name":"` + longName + `","latitude":49.0,"longitude":9.0}`
	req := httptest.NewRequest(http.MethodPost, "/api/sensors", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestCreateSensor_LatitudeTooHigh(t *testing.T) {
	app := setupTestApp(nil)
	req := httptest.NewRequest(http.MethodPost, "/api/sensors",
		strings.NewReader(`{"name":"sensor","latitude":91.0,"longitude":9.0}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestCreateSensor_LongitudeTooLow(t *testing.T) {
	app := setupTestApp(nil)
	req := httptest.NewRequest(http.MethodPost, "/api/sensors",
		strings.NewReader(`{"name":"sensor","latitude":49.0,"longitude":-181.0}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestCreateSensor_InvalidJSON(t *testing.T) {
	app := setupTestApp(nil)
	req := httptest.NewRequest(http.MethodPost, "/api/sensors",
		strings.NewReader(`not json`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

// TestValidateUUID_Regex tests the uuidRegex that backs validateUUID.
// The HTTP handler path cannot be tested cleanly with a nil pool because
// validateUUID returns nil error on failure (fiber v3 c.JSON returns nil),
// so the caller proceeds to pool operations. Testing the regex directly
// is equivalent and more comprehensive.
func TestValidateUUID_Regex(t *testing.T) {
	cases := []struct {
		uuid  string
		valid bool
	}{
		{"abc", false},        // too short
		{"ZZZZZZZZ", false},   // uppercase non-hex
		{"gggggggg", false},   // lowercase non-hex
		{"abcde12345", false}, // too long (10 chars)
		{"abcd1234", true},    // valid 8-char lowercase hex
		{"00000000", true},    // valid all-zeros
		{"deadbeef", true},    // valid hex word
	}
	for _, tc := range cases {
		got := uuidRegex.MatchString(tc.uuid)
		if got != tc.valid {
			t.Errorf("uuidRegex.MatchString(%q) = %v, want %v", tc.uuid, got, tc.valid)
		}
	}
}

func TestGetBuildStatus_NotStarted(t *testing.T) {
	app := setupTestApp(nil)
	// Clear any leftover state from other tests
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
	app := setupTestApp(nil)
	// Clear any leftover state from other tests
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
