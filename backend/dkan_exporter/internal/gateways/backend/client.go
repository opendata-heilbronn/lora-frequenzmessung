package dkan

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// Client is the main struct for interacting with the DKAN API.
type Client struct {
	BaseURL    string
	Username   string
	Password   string
	HTTPClient *http.Client
}

// NewClient creates a new DKAN client with a default 10-second timeout.
func NewClient(baseURL, username, password string) *Client {
	return &Client{
		BaseURL:  baseURL,
		Username: username,
		Password: password,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// do is a private helper that handles all the boilerplate for HTTP requests:
// JSON encoding/decoding, setting headers, and applying Basic Auth.
func (c *Client) do(ctx context.Context, method, endpoint string, bodyIn any, bodyOut any) error {
	reqURL := fmt.Sprintf("%s%s", c.BaseURL, endpoint)

	var reqBody io.Reader

	if bodyIn != nil {
		jsonData, err := json.Marshal(bodyIn)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}

		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set standard headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	req.Header.Set("Authorization", base64.StdEncoding.EncodeToString([]byte(c.Username+":"+c.Password)))

	// Apply Basic Authentication
	req.SetBasicAuth(c.Username, c.Password)

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer res.Body.Close()

	// Handle non-success HTTP status codes
	if res.StatusCode < 200 || res.StatusCode > 299 {
		errBody, _ := io.ReadAll(res.Body)
		return fmt.Errorf("API error (status %d): %s", res.StatusCode, string(errBody))
	}

	// Decode the response if a target struct was provided
	if bodyOut != nil {
		if err := json.NewDecoder(res.Body).Decode(bodyOut); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	slog.Info("noting to return")

	return nil
}
