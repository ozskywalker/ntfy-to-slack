package unit_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/ozskywalker/ntfy-to-slack/internal/config"
)


func TestNewMustachePostProcessor(t *testing.T) {
	tests := []struct {
		name         string
		template     string
		shouldError  bool
		errorContains string
	}{
		{
			name:        "valid template",
			template:    "Hello {{.Title}}: {{.Message}}",
			shouldError: false,
		},
		{
			name:        "empty template",
			template:    "",
			shouldError: false,
		},
		{
			name:         "invalid template syntax",
			template:     "Hello {{.Title",
			shouldError:  true,
			errorContains: "failed to parse template",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor, err := config.NewMustachePostProcessor(tt.template)
			
			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain %q, got: %v", tt.errorContains, err)
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			
			if processor == nil {
				t.Error("Expected processor but got nil")
			}
		})
	}
}

func TestNewMustachePostProcessorFromFile(t *testing.T) {
	// Create a temporary template file
	content := "Alert: {{.Title}}\nMessage: {{.Message}}"
	tmpFile, err := os.CreateTemp("", "template-*.tmpl")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	tests := []struct {
		name         string
		filePath     string
		shouldError  bool
		errorContains string
	}{
		{
			name:        "valid template file",
			filePath:    tmpFile.Name(),
			shouldError: false,
		},
		{
			name:         "non-existent file",
			filePath:     "non-existent-file.tmpl",
			shouldError:  true,
			errorContains: "failed to read template file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor, err := config.NewMustachePostProcessorFromFile(tt.filePath)
			
			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain %q, got: %v", tt.errorContains, err)
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			
			if processor == nil {
				t.Error("Expected processor but got nil")
			}
		})
	}
}

func TestMustachePostProcessor_Process(t *testing.T) {
	tests := []struct {
		name         string
		template     string
		message      *config.NtfyMessage
		expectedText string
		shouldError  bool
	}{
		{
			name:     "simple template with title and message",
			template: "{{.Title}}: {{.Message}}",
			message: &config.NtfyMessage{
				Title:   "Alert",
				Message: "System down",
			},
			expectedText: "Alert: System down",
		},
		{
			name:     "template with no title",
			template: "{{if .Title}}{{.Title}}: {{end}}{{.Message}}",
			message: &config.NtfyMessage{
				Title:   "",
				Message: "Simple message",
			},
			expectedText: "Simple message",
		},
		{
			name:     "template with all fields",
			template: "ID: {{.Id}}\nEvent: {{.Event}}\nTopic: {{.Topic}}\nTitle: {{.Title}}\nMessage: {{.Message}}\nTime: {{.Time}}",
			message: &config.NtfyMessage{
				Id:      "msg123",
				Event:   "message",
				Topic:   "alerts",
				Title:   "Test",
				Message: "Test message",
				Time:    1640995200,
			},
			expectedText: "ID: msg123\nEvent: message\nTopic: alerts\nTitle: Test\nMessage: Test message\nTime: 1640995200",
		},
		{
			name:     "halcyon-style formatting",
			template: "🚨 **{{.Title}}**\n{{.Message}}",
			message: &config.NtfyMessage{
				Title:   "Critical Alert",
				Message: "Service unavailable",
			},
			expectedText: "🚨 **Critical Alert**\nService unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor, err := config.NewMustachePostProcessor(tt.template)
			if err != nil {
				t.Fatalf("Failed to create processor: %v", err)
			}
			
			result, err := processor.Process(tt.message)
			
			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			
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

func TestNewWebhookPostProcessor(t *testing.T) {
	tests := []struct {
		name        string
		webhookURL  string
		httpClient  config.HTTPClient
	}{
		{
			name:       "with custom client",
			webhookURL: "https://example.com/webhook",
			httpClient: &MockHTTPClient{},
		},
		{
			name:       "with nil client",
			webhookURL: "https://example.com/webhook",
			httpClient: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := config.NewWebhookPostProcessor(tt.webhookURL, tt.httpClient)
			
			if processor == nil {
				t.Error("Expected processor but got nil")
			}
			
			// Note: webhookURL is not exported, so we can't test it directly
			// The fact that NewWebhookPostProcessor() doesn't panic and returns a non-nil processor is sufficient
		})
	}
}

func TestWebhookPostProcessor_Process(t *testing.T) {
	tests := []struct {
		name           string
		message        *config.NtfyMessage
		mockResponse   *http.Response
		mockError      error
		expectedText   string
		shouldError    bool
		errorContains  string
	}{
		{
			name: "successful JSON response",
			message: &config.NtfyMessage{
				Title:   "Test",
				Message: "Test message",
			},
			mockResponse: &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`{"text": "Formatted: Test - Test message"}`)),
			},
			expectedText: "Formatted: Test - Test message",
		},
		{
			name: "successful plain text response",
			message: &config.NtfyMessage{
				Title:   "Alert",
				Message: "System issue",
			},
			mockResponse: &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("Plain text response")),
			},
			expectedText: "Plain text response",
		},
		{
			name: "HTTP error status",
			message: &config.NtfyMessage{
				Title:   "Test",
				Message: "Test message",
			},
			mockResponse: &http.Response{
				StatusCode: 500,
				Body:       io.NopCloser(strings.NewReader("Internal server error")),
			},
			shouldError:   true,
			errorContains: "webhook returned status 500",
		},
		{
			name: "network error",
			message: &config.NtfyMessage{
				Title:   "Test",
				Message: "Test message",
			},
			mockError:     http.ErrHandlerTimeout,
			shouldError:   true,
			errorContains: "webhook request failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockHTTPClient{
				DoFunc: func(req *http.Request) (*http.Response, error) {
					// Verify request content
					if req.Method != http.MethodPost {
						t.Errorf("Expected POST method, got %s", req.Method)
					}
					
					if req.Header.Get("Content-Type") != "application/json" {
						t.Errorf("Expected Content-Type application/json, got %s", req.Header.Get("Content-Type"))
					}
					
					if req.Header.Get("User-Agent") != "ntfy-to-slack/2.0" {
						t.Errorf("Expected User-Agent ntfy-to-slack/2.0, got %s", req.Header.Get("User-Agent"))
					}
					
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					
					return tt.mockResponse, nil
				},
			}
			
			processor := config.NewWebhookPostProcessor("https://example.com/webhook", mockClient)
			result, err := processor.Process(tt.message)
			
			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain %q, got: %v", tt.errorContains, err)
				}
				return
			}
			
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

func TestWebhookPostProcessor_Integration(t *testing.T) {
	// Create a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json")
		}
		
		// Read and verify payload
		var msg config.NtfyMessage
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}
		
		// Return formatted response
		response := map[string]string{
			"text": "🔔 " + msg.Title + ": " + msg.Message,
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	processor := config.NewWebhookPostProcessor(server.URL, nil)
	
	message := &config.NtfyMessage{
		Title:   "Test Alert",
		Message: "This is a test",
	}
	
	result, err := processor.Process(message)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	expectedText := "🔔 Test Alert: This is a test"
	if result.Text != expectedText {
		t.Errorf("Expected text %q, got %q", expectedText, result.Text)
	}
}