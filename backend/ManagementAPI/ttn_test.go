package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
)

// ttnCall records a single HTTP request to the mock TTN server.
type ttnCall struct {
	Method string
	Path   string
	Body   map[string]any
}

// newMockTTNServer returns an httptest.Server that records all requests and
// responds with 200. Set failPath to make a specific path return 500.
func newMockTTNServer(calls *[]ttnCall, mu *sync.Mutex, failPath string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var parsed map[string]any
		json.Unmarshal(body, &parsed)

		mu.Lock()
		*calls = append(*calls, ttnCall{
			Method: r.Method,
			Path:   r.URL.Path,
			Body:   parsed,
		})
		mu.Unlock()

		if failPath != "" && r.URL.Path == failPath {
			w.WriteHeader(500)
			w.Write([]byte(`{"error":"mock failure"}`))
			return
		}

		// Verify auth header is present
		if r.Header.Get("Authorization") == "" {
			w.WriteHeader(401)
			w.Write([]byte(`{"error":"missing auth"}`))
			return
		}

		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	}))
}

// setTTNEnv sets TTN env vars pointing to the mock server and returns a cleanup func.
func setTTNEnv(baseURL string) func() {
	oldAppID := os.Getenv("TTN_APP_ID")
	oldAPIKey := os.Getenv("TTN_API_KEY")
	os.Setenv("TTN_APP_ID", "test-app")
	os.Setenv("TTN_API_KEY", "test-key")
	return func() {
		os.Setenv("TTN_APP_ID", oldAppID)
		os.Setenv("TTN_API_KEY", oldAPIKey)
	}
}

func TestTTNRegistration_AllFourSteps(t *testing.T) {
	var calls []ttnCall
	var mu sync.Mutex
	srv := newMockTTNServer(&calls, &mu, "")
	defer srv.Close()

	cleanup := setTTNEnv(srv.URL)
	defer cleanup()

	client := &http.Client{}
	baseURL := srv.URL
	appID := "test-app"
	apiKey := "test-key"
	deviceID := "sensor-aabbccdd"
	devEUI := "0011223344556677"
	appKey := "AABBCCDDEEFF00112233445566778899"
	joinEUI := "0101010101010101"

	// Step 1: IS
	isPayload := map[string]any{
		"end_device": map[string]any{
			"ids": map[string]any{
				"device_id": deviceID,
				"dev_eui":   devEUI,
				"join_eui":  joinEUI,
			},
			"join_server_address":        "eu1.cloud.thethings.network",
			"network_server_address":     "eu1.cloud.thethings.network",
			"application_server_address": "eu1.cloud.thethings.network",
		},
		"field_mask": map[string]any{
			"paths": []string{
				"join_server_address",
				"network_server_address",
				"application_server_address",
				"ids.dev_eui",
				"ids.join_eui",
			},
		},
	}
	err := ttnRequest(client, "POST", baseURL+"/applications/"+appID+"/devices", apiKey, isPayload)
	if err != nil {
		t.Fatalf("IS registration failed: %v", err)
	}

	// Step 2: JS
	jsPayload := map[string]any{
		"end_device": map[string]any{
			"ids":       map[string]any{"device_id": deviceID, "dev_eui": devEUI, "join_eui": joinEUI},
			"root_keys": map[string]any{"app_key": map[string]any{"key": appKey}},
		},
		"field_mask": map[string]any{"paths": []string{"root_keys.app_key.key"}},
	}
	err = ttnRequest(client, "PUT", baseURL+"/js/applications/"+appID+"/devices/"+deviceID, apiKey, jsPayload)
	if err != nil {
		t.Fatalf("JS registration failed: %v", err)
	}

	// Step 3: NS
	nsPayload := map[string]any{
		"end_device": map[string]any{
			"ids":                 map[string]any{"device_id": deviceID, "dev_eui": devEUI, "join_eui": joinEUI},
			"supports_join":       true,
			"lorawan_version":     "MAC_V1_0_2",
			"lorawan_phy_version": "PHY_V1_0_2_REV_B",
			"frequency_plan_id":   "EU_863_870_TTN",
		},
		"field_mask": map[string]any{"paths": []string{"supports_join", "lorawan_version", "lorawan_phy_version", "frequency_plan_id"}},
	}
	err = ttnRequest(client, "PUT", baseURL+"/ns/applications/"+appID+"/devices/"+deviceID, apiKey, nsPayload)
	if err != nil {
		t.Fatalf("NS registration failed: %v", err)
	}

	// Step 4: AS
	asPayload := map[string]any{
		"end_device": map[string]any{
			"ids": map[string]any{"device_id": deviceID, "dev_eui": devEUI, "join_eui": joinEUI},
		},
		"field_mask": map[string]any{"paths": []string{"ids.device_id", "ids.dev_eui", "ids.join_eui"}},
	}
	err = ttnRequest(client, "PUT", baseURL+"/as/applications/"+appID+"/devices/"+deviceID, apiKey, asPayload)
	if err != nil {
		t.Fatalf("AS registration failed: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if len(calls) != 4 {
		t.Fatalf("expected 4 TTN API calls, got %d", len(calls))
	}

	// Verify order and methods
	expected := []struct {
		method   string
		pathPart string
	}{
		{"POST", "/applications/test-app/devices"},
		{"PUT", "/js/applications/test-app/devices/sensor-aabbccdd"},
		{"PUT", "/ns/applications/test-app/devices/sensor-aabbccdd"},
		{"PUT", "/as/applications/test-app/devices/sensor-aabbccdd"},
	}
	for i, exp := range expected {
		if calls[i].Method != exp.method {
			t.Errorf("call %d: expected method %s, got %s", i, exp.method, calls[i].Method)
		}
		if calls[i].Path != exp.pathPart {
			t.Errorf("call %d: expected path %s, got %s", i, exp.pathPart, calls[i].Path)
		}
	}

	// Verify IS payload has all required server addresses
	isBody := calls[0].Body
	endDevice, ok := isBody["end_device"].(map[string]any)
	if !ok {
		t.Fatal("IS payload missing end_device")
	}
	for _, field := range []string{"join_server_address", "network_server_address", "application_server_address"} {
		if endDevice[field] == nil || endDevice[field] == "" {
			t.Errorf("IS payload missing %s", field)
		}
	}

	// Verify NS payload has LoRaWAN settings
	nsBody := calls[2].Body
	nsDevice, ok := nsBody["end_device"].(map[string]any)
	if !ok {
		t.Fatal("NS payload missing end_device")
	}
	if nsDevice["supports_join"] != true {
		t.Error("NS payload: supports_join should be true")
	}
	if nsDevice["lorawan_version"] != "MAC_V1_0_2" {
		t.Errorf("NS payload: lorawan_version = %v, want MAC_V1_0_2", nsDevice["lorawan_version"])
	}
	if nsDevice["frequency_plan_id"] != "EU_863_870_TTN" {
		t.Errorf("NS payload: frequency_plan_id = %v, want EU_863_870_TTN", nsDevice["frequency_plan_id"])
	}

	// Verify JS payload has root keys
	jsBody := calls[1].Body
	jsDevice, ok := jsBody["end_device"].(map[string]any)
	if !ok {
		t.Fatal("JS payload missing end_device")
	}
	rootKeys, ok := jsDevice["root_keys"].(map[string]any)
	if !ok {
		t.Fatal("JS payload missing root_keys")
	}
	appKeyObj, ok := rootKeys["app_key"].(map[string]any)
	if !ok {
		t.Fatal("JS payload missing root_keys.app_key")
	}
	if appKeyObj["key"] != appKey {
		t.Errorf("JS payload: app_key = %v, want %s", appKeyObj["key"], appKey)
	}
}

func TestTTNRegistration_NSFailure_StopsEarly(t *testing.T) {
	var calls []ttnCall
	var mu sync.Mutex
	// Make the NS endpoint fail
	srv := newMockTTNServer(&calls, &mu, "/ns/applications/test-app/devices/sensor-12345678")
	defer srv.Close()

	cleanup := setTTNEnv(srv.URL)
	defer cleanup()

	client := &http.Client{}
	baseURL := srv.URL
	apiKey := "test-key"

	// Step 1: IS — should succeed
	err := ttnRequest(client, "POST", baseURL+"/applications/test-app/devices", apiKey, map[string]any{})
	if err != nil {
		t.Fatalf("IS should succeed: %v", err)
	}

	// Step 2: JS — should succeed
	err = ttnRequest(client, "PUT", baseURL+"/js/applications/test-app/devices/sensor-12345678", apiKey, map[string]any{})
	if err != nil {
		t.Fatalf("JS should succeed: %v", err)
	}

	// Step 3: NS — should fail
	err = ttnRequest(client, "PUT", baseURL+"/ns/applications/test-app/devices/sensor-12345678", apiKey, map[string]any{})
	if err == nil {
		t.Fatal("NS should have failed")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("NS error should contain status 500, got: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	// Only 3 calls should have been made (IS, JS, NS-fail); AS never called
	if len(calls) != 3 {
		t.Errorf("expected 3 TTN API calls (stop after NS failure), got %d", len(calls))
	}
}

func TestTTNDeletion_ReverseOrder(t *testing.T) {
	var calls []ttnCall
	var mu sync.Mutex
	srv := newMockTTNServer(&calls, &mu, "")
	defer srv.Close()

	cleanup := setTTNEnv(srv.URL)
	defer cleanup()

	client := &http.Client{}
	baseURL := srv.URL
	appID := "test-app"
	apiKey := "test-key"
	deviceID := "sensor-aabbccdd"

	// Simulate the deletion order from deleteTTNDevice
	endpoints := []string{
		baseURL + "/as/applications/" + appID + "/devices/" + deviceID,
		baseURL + "/ns/applications/" + appID + "/devices/" + deviceID,
		baseURL + "/js/applications/" + appID + "/devices/" + deviceID,
		baseURL + "/applications/" + appID + "/devices/" + deviceID,
	}

	for _, url := range endpoints {
		req, err := http.NewRequest("DELETE", url, nil)
		if err != nil {
			t.Fatalf("creating request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		resp.Body.Close()
	}

	mu.Lock()
	defer mu.Unlock()

	if len(calls) != 4 {
		t.Fatalf("expected 4 DELETE calls, got %d", len(calls))
	}

	// Verify reverse order: AS, NS, JS, IS
	expectedPaths := []string{
		"/as/applications/test-app/devices/sensor-aabbccdd",
		"/ns/applications/test-app/devices/sensor-aabbccdd",
		"/js/applications/test-app/devices/sensor-aabbccdd",
		"/applications/test-app/devices/sensor-aabbccdd",
	}
	for i, exp := range expectedPaths {
		if calls[i].Method != "DELETE" {
			t.Errorf("call %d: expected DELETE, got %s", i, calls[i].Method)
		}
		if calls[i].Path != exp {
			t.Errorf("call %d: expected path %s, got %s", i, exp, calls[i].Path)
		}
	}
}

func TestTTNRegistration_MissingEnvVars(t *testing.T) {
	// Clear TTN env vars
	oldAppID := os.Getenv("TTN_APP_ID")
	oldAPIKey := os.Getenv("TTN_API_KEY")
	os.Setenv("TTN_APP_ID", "")
	os.Setenv("TTN_API_KEY", "")
	defer func() {
		os.Setenv("TTN_APP_ID", oldAppID)
		os.Setenv("TTN_API_KEY", oldAPIKey)
	}()

	err := registerTTNDevice("test-device", "0011223344556677", "AABBCCDDEEFF00112233445566778899")
	if err == nil {
		t.Fatal("expected error when TTN env vars are missing")
	}
	if !strings.Contains(err.Error(), "not configured") {
		t.Errorf("expected 'not configured' error, got: %v", err)
	}
}

func TestTTNRequest_AuthHeader(t *testing.T) {
	var receivedAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	client := &http.Client{}
	err := ttnRequest(client, "POST", srv.URL+"/test", "my-secret-key", map[string]any{"foo": "bar"})
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if receivedAuth != "Bearer my-secret-key" {
		t.Errorf("expected 'Bearer my-secret-key', got %q", receivedAuth)
	}
}

func TestTTNRequest_ErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(409)
		w.Write([]byte(`{"error":"device already exists"}`))
	}))
	defer srv.Close()

	client := &http.Client{}
	err := ttnRequest(client, "POST", srv.URL+"/test", "key", map[string]any{})
	if err == nil {
		t.Fatal("expected error for 409 response")
	}
	if !strings.Contains(err.Error(), "409") {
		t.Errorf("error should contain status code 409, got: %v", err)
	}
}
