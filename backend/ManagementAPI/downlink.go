package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Downlink command IDs sent to sensors via TTN.
const (
	CmdRequestVersion byte = 0x01
	// Future: CmdOTAUpdate byte = 0x02
)

// sendDownlink schedules a downlink message to a TTN device via the AS push endpoint.
// command is the first byte; payload (optional) is appended after it.
func sendDownlink(ttnBaseURL, appID, apiKey, deviceID string, command byte, payload []byte) error {
	frame := append([]byte{command}, payload...)
	encoded := base64.StdEncoding.EncodeToString(frame)

	body := map[string]any{
		"downlinks": []map[string]any{
			{
				"f_port":      2,
				"frm_payload": encoded,
				"priority":    "NORMAL",
			},
		},
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/as/applications/%s/devices/%s/down/push", ttnBaseURL, appID, deviceID)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errBody map[string]any
		json.NewDecoder(resp.Body).Decode(&errBody)
		return fmt.Errorf("TTN downlink HTTP %d: %v", resp.StatusCode, errBody)
	}
	return nil
}
