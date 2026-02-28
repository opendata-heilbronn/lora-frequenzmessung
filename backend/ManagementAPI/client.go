package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

func internalURL(path string) string {
	base := os.Getenv("BACKEND_INTERNAL_URL")
	if base == "" {
		base = "http://localhost:3001"
	}
	return fmt.Sprintf("%s%s", base, path)
}

func internalRequest(method, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, internalURL(path), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Internal-Key", os.Getenv("INTERNAL_API_KEY"))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	client := &http.Client{Timeout: 30 * time.Second}
	return client.Do(req)
}
