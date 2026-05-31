package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

const ttnServerHost = "eu1.cloud.thethings.network"

func getTTNConfig() (appID, apiKey, baseURL string) {
	appID = os.Getenv("TTN_APP_ID")
	apiKey = os.Getenv("TTN_API_KEY")
	baseURL = "https://eu1.cloud.thethings.network/api/v3"

	return
}

// registerTTNDevice registers an OTAA device in TTN v3.
// devEUI and appKey must be uppercase hex strings (no separators).
//
// TTN v3 requires four registration steps:
//  1. Identity Server (IS) — create the device record
//  2. Join Server (JS)     — set OTAA root keys
//  3. Network Server (NS)  — configure LoRaWAN version & frequency plan
//  4. Application Server (AS) — register on the application layer
func registerTTNDevice(deviceID, devEUI, appKey string) error {
	appID, apiKey, baseURL := getTTNConfig()
	if appID == "" || apiKey == "" {
		return fmt.Errorf("TTN_APP_ID or TTN_API_KEY not configured")
	}

	joinEUI := "0101010101010101"
	client := &http.Client{}

	// Step 1: Create device in Identity Server (IS)
	isPayload := map[string]any{
		"end_device": map[string]any{
			"ids": map[string]any{
				"device_id": deviceID,
				"dev_eui":   devEUI,
				"join_eui":  joinEUI,
			},
			"join_server_address":        ttnServerHost,
			"network_server_address":     ttnServerHost,
			"application_server_address": ttnServerHost,
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

	// Step 2: Set OTAA root keys in Join Server (JS)
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

	// Step 3: Register on Network Server (NS) — LoRaWAN settings
	nsPayload := map[string]any{
		"end_device": map[string]any{
			"ids": map[string]any{
				"device_id": deviceID,
				"dev_eui":   devEUI,
				"join_eui":  joinEUI,
			},
			"supports_join":       true,
			"lorawan_version":     "MAC_V1_0_2",
			"lorawan_phy_version": "PHY_V1_0_2_REV_B",
			"frequency_plan_id":   "EU_863_870_TTN",
		},
		"field_mask": map[string]any{
			"paths": []string{
				"supports_join",
				"lorawan_version",
				"lorawan_phy_version",
				"frequency_plan_id",
			},
		},
	}

	if err := ttnRequest(client, "PUT",
		fmt.Sprintf("%s/ns/applications/%s/devices/%s", baseURL, appID, deviceID),
		apiKey, nsPayload); err != nil {
		return fmt.Errorf("NS registration: %w", err)
	}

	// Step 4: Register on Application Server (AS)
	asPayload := map[string]any{
		"end_device": map[string]any{
			"ids": map[string]any{
				"device_id": deviceID,
				"dev_eui":   devEUI,
				"join_eui":  joinEUI,
			},
		},
		"field_mask": map[string]any{
			"paths": []string{
				"ids.device_id",
				"ids.dev_eui",
				"ids.join_eui",
			},
		},
	}

	if err := ttnRequest(client, "PUT",
		fmt.Sprintf("%s/as/applications/%s/devices/%s", baseURL, appID, deviceID),
		apiKey, asPayload); err != nil {
		return fmt.Errorf("AS registration: %w", err)
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

	// Delete from AS, NS, JS first, then IS (reverse order of creation).
	// Best effort — continue even if individual deletes fail.
	endpoints := []string{
		fmt.Sprintf("%s/as/applications/%s/devices/%s", baseURL, appID, deviceID),
		fmt.Sprintf("%s/ns/applications/%s/devices/%s", baseURL, appID, deviceID),
		fmt.Sprintf("%s/js/applications/%s/devices/%s", baseURL, appID, deviceID),
		fmt.Sprintf("%s/applications/%s/devices/%s", baseURL, appID, deviceID),
	}

	var lastErr error

	for _, url := range endpoints {
		req, err := http.NewRequest("DELETE", url, http.NoBody)
		if err != nil {
			lastErr = err
			continue
		}

		req.Header.Set("Authorization", "Bearer "+apiKey)

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		resp.Body.Close()
	}

	return lastErr
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
