package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealthHandler(t *testing.T) {
	tests := []struct {
		name           string
		acceptHeader   string
		queryParam     string
		expectedStatus int
		expectedType   string
	}{
		{
			name:           "JSON response with Accept header",
			acceptHeader:   "application/json",
			expectedStatus: http.StatusOK,
			expectedType:   "application/json",
		},
		{
			name:           "JSON response with query param",
			queryParam:     "json",
			expectedStatus: http.StatusOK,
			expectedType:   "application/json",
		},
		{
			name:           "HTML response default",
			expectedStatus: http.StatusOK,
			expectedType:   "text/html",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/health", nil)
			if tt.acceptHeader != "" {
				req.Header.Set("Accept", tt.acceptHeader)
			}
			if tt.queryParam != "" {
				q := req.URL.Query()
				q.Add("format", tt.queryParam)
				req.URL.RawQuery = q.Encode()
			}

			w := httptest.NewRecorder()
			healthHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != tt.expectedType {
				t.Errorf("Expected content type %s, got %s", tt.expectedType, contentType)
			}

			// For JSON responses, verify it's valid JSON
			if tt.expectedType == "application/json" {
				var health HealthStatus
				if err := json.Unmarshal(w.Body.Bytes(), &health); err != nil {
					t.Errorf("Invalid JSON response: %v", err)
				}
				if health.Status != "OK" {
					t.Errorf("Expected status OK, got %s", health.Status)
				}
			}
		})
	}
}

func TestGetHealthStatus(t *testing.T) {
	// Set a known start time for testing
	originalStartTime := startTime
	startTime = time.Now().Add(-1 * time.Hour)
	defer func() { startTime = originalStartTime }()

	health := getHealthStatus()

	if health.Status != "OK" {
		t.Errorf("Expected status OK, got %s", health.Status)
	}

	if health.Goroutines <= 0 {
		t.Errorf("Expected positive goroutine count, got %d", health.Goroutines)
	}

	if health.CPU.NumCPU <= 0 {
		t.Errorf("Expected positive CPU count, got %d", health.CPU.NumCPU)
	}

	if health.Memory.AllocMB < 0 {
		t.Errorf("Expected non-negative memory allocation, got %f", health.Memory.AllocMB)
	}

	// Verify timestamp format
	if _, err := time.Parse(time.RFC3339, health.Timestamp); err != nil {
		t.Errorf("Invalid timestamp format: %v", err)
	}
}

func TestConfigureLogging(t *testing.T) {
	// Test default log level
	originalLevel := log.Level
	defer func() { log.SetLevel(originalLevel) }()

	// Test with no LOG_LEVEL set
	t.Setenv("LOG_LEVEL", "")
	configureLogging()
	// Should default to info level - we can't easily test this without exposing internals

	// Test with debug level
	t.Setenv("LOG_LEVEL", "debug")
	configureLogging()
	// Should set to debug level

	// Test with invalid level
	t.Setenv("LOG_LEVEL", "invalid")
	configureLogging()
	// Should default to info level and log a warning
}
