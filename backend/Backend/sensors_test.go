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

	app.Post("/internal/sensors", createSensor(pool, ctx))

	return app
}

func TestCreateSensor_EmptyName(t *testing.T) {
	app := setupTestApp(nil)
	req := httptest.NewRequest(http.MethodPost, "/internal/sensors",
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
	req := httptest.NewRequest(http.MethodPost, "/internal/sensors", strings.NewReader(body))
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
	req := httptest.NewRequest(http.MethodPost, "/internal/sensors",
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
	req := httptest.NewRequest(http.MethodPost, "/internal/sensors",
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
	req := httptest.NewRequest(http.MethodPost, "/internal/sensors",
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
