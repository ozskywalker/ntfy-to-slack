package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ozskywalker/ntfy-to-slack/internal/config"
)

// TestWebhookPostProcessor_RetryMechanism tests the retry logic with exponential backoff
func TestWebhookPostProcessor_RetryMechanism(t *testing.T) {
	tests := []struct {
		name                string
		failureAttempts     int
		totalAttempts      int
		expectedRetries    int
		shouldSucceed      bool
		responseDelay      time.Duration
	}{
		{
			name:            "succeed on first attempt",
			failureAttempts: 0,
			totalAttempts:  1,
			expectedRetries: 0,
			shouldSucceed:   true,
		},
		{
			name:            "succeed on second attempt",
			failureAttempts: 1,
			totalAttempts:  2,
			expectedRetries: 1,
			shouldSucceed:   true,
		},
		{
			name:            "fail after all retries",
			failureAttempts: 5,
			totalAttempts:  4, // 1 initial + 3 retries
			expectedRetries: 3,
			shouldSucceed:   false,
		},
		{
			name:            "succeed after maximum retries",
			failureAttempts: 3,
			totalAttempts:  4, // 1 initial + 3 retries
			expectedRetries: 3,
			shouldSucceed:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var attemptCount int32
			
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				attempt := atomic.AddInt32(&attemptCount, 1)
				
				// Add response delay if specified
				if tt.responseDelay > 0 {
					time.Sleep(tt.responseDelay)
				}
				
				if int(attempt) <= tt.failureAttempts {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte("Server error"))
					return
				}
				
				// Success response
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]string{
					"text": "Success after retries",
				})
			}))
			defer server.Close()
			
			processor := config.NewWebhookPostProcessorWithConfig(server.URL, 5, 3, 1)
			
			message := &config.NtfyMessage{
				Title:   "Test",
				Message: "Retry test",
			}
			
			result, err := processor.Process(message)
			
			if tt.shouldSucceed {
				if err != nil {
					t.Errorf("Expected success but got error: %v", err)
				}
				if result == nil || result.Text != "Success after retries" {
					t.Errorf("Expected success message, got: %v", result)
				}
			} else {
				if err == nil {
					t.Error("Expected error but got success")
				}
				if !strings.Contains(err.Error(), "failed after") {
					t.Errorf("Expected retry failure message, got: %v", err)
				}
			}
			
			if int(attemptCount) != tt.totalAttempts {
				t.Errorf("Expected %d total attempts, got %d", tt.totalAttempts, attemptCount)
			}
		})
	}
}

// TestWebhookPostProcessor_ResponseSizeLimits tests response size limitation
func TestWebhookPostProcessor_ResponseSizeLimits(t *testing.T) {
	tests := []struct {
		name              string
		responseSizeMB    int
		maxResponseSizeMB int
		shouldTruncate    bool
	}{
		{
			name:              "response within limit",
			responseSizeMB:    1,
			maxResponseSizeMB: 2,
			shouldTruncate:    false,
		},
		{
			name:              "response at limit",
			responseSizeMB:    1,
			maxResponseSizeMB: 1,
			shouldTruncate:    false,
		},
		{
			name:              "response exceeds limit",
			responseSizeMB:    2,
			maxResponseSizeMB: 1,
			shouldTruncate:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Generate response of specified size
				responseSize := tt.responseSizeMB * 1024 * 1024
				largeResponse := strings.Repeat("a", responseSize)
				
				w.Header().Set("Content-Type", "text/plain")
				w.Write([]byte(largeResponse))
			}))
			defer server.Close()
			
			processor := config.NewWebhookPostProcessorWithConfig(server.URL, 5, 0, tt.maxResponseSizeMB)
			
			message := &config.NtfyMessage{
				Title:   "Test",
				Message: "Size test",
			}
			
			result, err := processor.Process(message)
			
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			
			if result == nil {
				t.Error("Expected result but got nil")
				return
			}
			
			expectedMaxSize := tt.maxResponseSizeMB * 1024 * 1024
			if tt.shouldTruncate {
				if len(result.Text) != expectedMaxSize {
					t.Errorf("Expected response to be truncated to %d bytes, got %d", expectedMaxSize, len(result.Text))
				}
			} else {
				originalSize := tt.responseSizeMB * 1024 * 1024
				if len(result.Text) != originalSize {
					t.Errorf("Expected response size %d bytes, got %d", originalSize, len(result.Text))
				}
			}
		})
	}
}

// TestWebhookPostProcessor_TimeoutHandling tests timeout scenarios
func TestWebhookPostProcessor_TimeoutHandling(t *testing.T) {
	tests := []struct {
		name           string
		serverDelay    time.Duration
		timeoutSeconds int
		shouldTimeout  bool
	}{
		{
			name:           "request within timeout",
			serverDelay:    100 * time.Millisecond,
			timeoutSeconds: 2,
			shouldTimeout:  false,
		},
		{
			name:           "request exceeds timeout",
			serverDelay:    3 * time.Second,
			timeoutSeconds: 1,
			shouldTimeout:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(tt.serverDelay)
				json.NewEncoder(w).Encode(map[string]string{
					"text": "Delayed response",
				})
			}))
			defer server.Close()
			
			processor := config.NewWebhookPostProcessorWithConfig(server.URL, tt.timeoutSeconds, 0, 1)
			
			message := &config.NtfyMessage{
				Title:   "Test",
				Message: "Timeout test",
			}
			
			start := time.Now()
			result, err := processor.Process(message)
			duration := time.Since(start)
			
			if tt.shouldTimeout {
				if err == nil {
					t.Error("Expected timeout error but got success")
				}
				// Should timeout within reasonable bounds
				expectedTimeout := time.Duration(tt.timeoutSeconds) * time.Second
				if duration > expectedTimeout+500*time.Millisecond {
					t.Errorf("Timeout took too long: %v, expected around %v", duration, expectedTimeout)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result == nil || result.Text != "Delayed response" {
					t.Errorf("Expected success message, got: %v", result)
				}
			}
		})
	}
}

// TestWebhookPostProcessor_ErrorStatusCodes tests different HTTP status code handling
func TestWebhookPostProcessor_ErrorStatusCodes(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		shouldRetry    bool
		maxRetries     int
		expectedAttempts int
	}{
		{
			name:           "client error 400 - no retry",
			statusCode:     400,
			shouldRetry:    false,
			maxRetries:     3,
			expectedAttempts: 1,
		},
		{
			name:           "client error 404 - no retry",
			statusCode:     404,
			shouldRetry:    false,
			maxRetries:     3,
			expectedAttempts: 1,
		},
		{
			name:           "server error 500 - retry",
			statusCode:     500,
			shouldRetry:    true,
			maxRetries:     2,
			expectedAttempts: 3, // 1 initial + 2 retries
		},
		{
			name:           "server error 503 - retry",
			statusCode:     503,
			shouldRetry:    true,
			maxRetries:     1,
			expectedAttempts: 2, // 1 initial + 1 retry
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var attemptCount int32
			
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&attemptCount, 1)
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(fmt.Sprintf("Error %d", tt.statusCode)))
			}))
			defer server.Close()
			
			processor := config.NewWebhookPostProcessorWithConfig(server.URL, 5, tt.maxRetries, 1)
			
			message := &config.NtfyMessage{
				Title:   "Test",
				Message: "Status code test",
			}
			
			result, err := processor.Process(message)
			
			// Should always fail
			if err == nil {
				t.Error("Expected error but got success")
			}
			
			if result != nil {
				t.Error("Expected nil result on error")
			}
			
			if int(attemptCount) != tt.expectedAttempts {
				t.Errorf("Expected %d attempts, got %d", tt.expectedAttempts, attemptCount)
			}
			
			if !strings.Contains(err.Error(), fmt.Sprintf("status %d", tt.statusCode)) {
				t.Errorf("Expected error to mention status code %d, got: %v", tt.statusCode, err)
			}
		})
	}
}

// TestMustachePostProcessor_TemplateValidation tests template syntax validation
func TestMustachePostProcessor_TemplateValidation(t *testing.T) {
	tests := []struct {
		name          string
		template      string
		shouldSucceed bool
		errorContains string
	}{
		{
			name:          "valid template",
			template:      "{{.Title}}: {{.Message}}",
			shouldSucceed: true,
		},
		{
			name:          "template with conditionals",
			template:      "{{if .Title}}{{.Title}}: {{end}}{{.Message}}",
			shouldSucceed: true,
		},
		{
			name:          "template with range on non-existent field",
			template:      "{{range .NonExistentField}}{{.}}{{end}}",
			shouldSucceed: false, // Should fail validation due to field access
			errorContains: "template validation failed",
		},
		{
			name:          "template with invalid syntax",
			template:      "{{.Title",
			shouldSucceed: false,
			errorContains: "failed to parse template",
		},
		{
			name:          "template with invalid field access",
			template:      "{{.Title.NonExistentSubField}}",
			shouldSucceed: false,
			errorContains: "template validation failed",
		},
		{
			name:          "empty template",
			template:      "",
			shouldSucceed: true,
		},
		{
			name:          "template with functions",
			template:      "{{.Title | printf \"%s\"}}",
			shouldSucceed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor, err := config.NewMustachePostProcessor(tt.template)
			
			if tt.shouldSucceed {
				if err != nil {
					t.Errorf("Expected success but got error: %v", err)
				}
				if processor == nil {
					t.Error("Expected processor but got nil")
				}
			} else {
				if err == nil {
					t.Error("Expected error but got success")
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain %q, got: %v", tt.errorContains, err)
				}
			}
		})
	}
}

// TestMustachePostProcessor_TemplateFileValidation tests template file loading with validation
func TestMustachePostProcessor_TemplateFileValidation(t *testing.T) {
	// Create temporary files for testing
	validTemplate := "🚨 {{.Title}}\n📄 {{.Message}}"
	invalidTemplate := "{{.Title"
	
	validFile := createTempFile(t, "valid-*.tmpl", validTemplate)
	defer os.Remove(validFile)
	
	invalidFile := createTempFile(t, "invalid-*.tmpl", invalidTemplate)
	defer os.Remove(invalidFile)
	
	tests := []struct {
		name          string
		filePath      string
		shouldSucceed bool
		errorContains string
	}{
		{
			name:          "valid template file",
			filePath:      validFile,
			shouldSucceed: true,
		},
		{
			name:          "invalid template file",
			filePath:      invalidFile,
			shouldSucceed: false,
			errorContains: "failed to parse template",
		},
		{
			name:          "non-existent file",
			filePath:      "non-existent-file.tmpl",
			shouldSucceed: false,
			errorContains: "failed to read template file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor, err := config.NewMustachePostProcessorFromFile(tt.filePath)
			
			if tt.shouldSucceed {
				if err != nil {
					t.Errorf("Expected success but got error: %v", err)
				}
				if processor == nil {
					t.Error("Expected processor but got nil")
				}
			} else {
				if err == nil {
					t.Error("Expected error but got success")
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain %q, got: %v", tt.errorContains, err)
				}
			}
		})
	}
}

// TestWebhookPostProcessor_MalformedResponse tests handling of malformed webhook responses
func TestWebhookPostProcessor_MalformedResponse(t *testing.T) {
	tests := []struct {
		name         string
		response     string
		contentType  string
		expectedText string
	}{
		{
			name:         "valid JSON response",
			response:     `{"text": "Valid JSON"}`,
			contentType:  "application/json",
			expectedText: "Valid JSON",
		},
		{
			name:         "invalid JSON - treated as text",
			response:     `{"text": "Invalid JSON"`,
			contentType:  "application/json",
			expectedText: `{"text": "Invalid JSON"`,
		},
		{
			name:         "plain text response",
			response:     "Plain text message",
			contentType:  "text/plain",
			expectedText: "Plain text message",
		},
		{
			name:         "empty response",
			response:     "",
			contentType:  "application/json",
			expectedText: "",
		},
		{
			name:         "HTML response",
			response:     "<html><body>HTML content</body></html>",
			contentType:  "text/html",
			expectedText: "<html><body>HTML content</body></html>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tt.contentType)
				w.Write([]byte(tt.response))
			}))
			defer server.Close()
			
			processor := config.NewWebhookPostProcessorWithConfig(server.URL, 5, 0, 1)
			
			message := &config.NtfyMessage{
				Title:   "Test",
				Message: "Response test",
			}
			
			result, err := processor.Process(message)
			
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			
			if result == nil {
				t.Error("Expected result but got nil")
				return
			}
			
			if result.Text != tt.expectedText {
				t.Errorf("Expected text %q, got %q", tt.expectedText, result.Text)
			}
		})
	}
}

// Helper function to create temporary files for testing
func createTempFile(t *testing.T, pattern, content string) string {
	tmpFile, err := os.CreateTemp("", pattern)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	
	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	
	tmpFile.Close()
	return tmpFile.Name()
}

// TestWebhookPostProcessor_ConcurrentRequests tests concurrent webhook processing
func TestWebhookPostProcessor_ConcurrentRequests(t *testing.T) {
	var requestCount int32
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)
		
		// Add small delay to simulate processing
		time.Sleep(10 * time.Millisecond)
		
		json.NewEncoder(w).Encode(map[string]string{
			"text": fmt.Sprintf("Response %d", count),
		})
	}))
	defer server.Close()
	
	processor := config.NewWebhookPostProcessorWithConfig(server.URL, 5, 0, 1)
	
	const concurrency = 10
	results := make(chan *config.SlackMessage, concurrency)
	errors := make(chan error, concurrency)
	
	// Launch concurrent requests
	for i := 0; i < concurrency; i++ {
		go func(id int) {
			message := &config.NtfyMessage{
				Title:   fmt.Sprintf("Test %d", id),
				Message: "Concurrent test",
			}
			
			result, err := processor.Process(message)
			if err != nil {
				errors <- err
			} else {
				results <- result
			}
		}(i)
	}
	
	// Collect results
	var successCount, errorCount int
	for i := 0; i < concurrency; i++ {
		select {
		case <-results:
			successCount++
		case err := <-errors:
			errorCount++
			t.Errorf("Unexpected error in concurrent request: %v", err)
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for concurrent requests")
		}
	}
	
	if successCount != concurrency {
		t.Errorf("Expected %d successful requests, got %d", concurrency, successCount)
	}
	
	if errorCount > 0 {
		t.Errorf("Expected 0 errors, got %d", errorCount)
	}
	
	if int(requestCount) != concurrency {
		t.Errorf("Expected %d server requests, got %d", concurrency, requestCount)
	}
}