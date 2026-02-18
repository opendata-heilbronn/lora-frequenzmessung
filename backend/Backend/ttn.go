package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

func getTTNConfig() (appID, apiKey, baseURL string) {
	appID = os.Getenv("TTN_APP_ID")
	apiKey = os.Getenv("TTN_API_KEY")
	baseURL = "https://eu1.cloud.thethings.network/api/v3"
	return
}

// registerTTNDevice registers an OTAA device in TTN v3.
// devEUI and appKey must be uppercase hex strings (no separators).
func registerTTNDevice(deviceID, devEUI, appKey string) error {
	appID, apiKey, baseURL := getTTNConfig()
	if appID == "" || apiKey == "" {
		return fmt.Errorf("TTN_APP_ID or TTN_API_KEY not configured")
	}

	joinEUI := "0101010101010101"
	client := &http.Client{}

	// Step 1: Create device in Identity Server
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

	if err := ttnRequest(client, "POST",
		fmt.Sprintf("%s/applications/%s/devices", baseURL, appID),
		apiKey, isPayload); err != nil {
		return fmt.Errorf("IS registration: %w", err)
	}

	// Step 2: Set OTAA root keys in Join Server
	jsPayload := map[string]any{
		"end_device": map[string]any{
			"ids": map[string]any{
				"device_id": deviceID,
				"dev_eui":   devEUI,
				"join_eui":  joinEUI,
			},
			"root_keys": map[string]any{
				"app_key": map[string]any{
					"key": appKey,
				},
			},
		},
		"field_mask": map[string]any{
			"paths": []string{"root_keys.app_key.key"},
		},
	}

	if err := ttnRequest(client, "PUT",
		fmt.Sprintf("%s/js/applications/%s/devices/%s", baseURL, appID, deviceID),
		apiKey, jsPayload); err != nil {
		return fmt.Errorf("JS key registration: %w", err)
	}

	return nil
}

// deleteTTNDevice removes a device from TTN (best effort).
func deleteTTNDevice(deviceID string) error {
	appID, apiKey, baseURL := getTTNConfig()
	if appID == "" || apiKey == "" {
		return nil
	}

	client := &http.Client{}
	url := fmt.Sprintf("%s/applications/%s/devices/%s", baseURL, appID, deviceID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func ttnRequest(client *http.Client, method, url, apiKey string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errBody map[string]any
		json.NewDecoder(resp.Body).Decode(&errBody)
		return fmt.Errorf("HTTP %d: %v", resp.StatusCode, errBody)
	}
	return nil
}
