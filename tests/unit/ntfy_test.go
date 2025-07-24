package unit_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ozskywalker/ntfy-to-slack/internal/config"
	"github.com/ozskywalker/ntfy-to-slack/internal/ntfy"
)


func TestNewNtfyClient(t *testing.T) {
	tests := []struct {
		name   string
		domain string
		topic  string
		auth   string
		client config.HTTPClient
	}{
		{
			name:   "with custom client",
			domain: "ntfy.sh",
			topic:  "test-topic",
			auth:   "auth-token",
			client: &MockHTTPClient{},
		},
		{
			name:   "with nil client creates default",
			domain: "ntfy.sh", 
			topic:  "test-topic",
			auth:   "",
			client: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := ntfy.NewClient(tt.domain, tt.topic, tt.auth, tt.client)
			
			if client == nil {
				t.Error("ntfy.NewClient() returned nil")
			}
			
			// Note: internal fields are not exported, so we can't test them directly
			// The fact that NewClient() doesn't panic and returns a non-nil client is sufficient
		})
	}
}

func TestNtfyHTTPClient_Connect(t *testing.T) {
	tests := []struct {
		name         string
		domain       string
		topic        string
		auth         string
		mockResponse *http.Response
		mockError    error
		wantErr      bool
		validateReq  bool
	}{
		{
			name:   "successful connection",
			domain: "ntfy.sh",
			topic:  "test-topic",
			auth:   "",
			mockResponse: &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("")),
			},
			wantErr:     false,
			validateReq: true,
		},
		{
			name:   "successful connection with auth",
			domain: "ntfy.sh",
			topic:  "test-topic",
			auth:   "auth-token",
			mockResponse: &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("")),
			},
			wantErr:     false,
			validateReq: true,
		},
		{
			name:    "invalid domain",
			domain:  "invalid-domain",
			topic:   "test-topic",
			auth:    "",
			wantErr: true,
		},
		{
			name:    "invalid topic",
			domain:  "ntfy.sh",
			topic:   "invalid topic with spaces",
			auth:    "",
			wantErr: true,
		},
		{
			name:      "HTTP client error",
			domain:    "ntfy.sh",
			topic:     "test-topic",
			auth:      "",
			mockError: errors.New("network error"),
			wantErr:   true,
		},
		{
			name:   "HTTP 404 error",
			domain: "ntfy.sh",
			topic:  "test-topic",
			auth:   "",
			mockResponse: &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(strings.NewReader("")),
			},
			wantErr: true,
		},
		{
			name:   "HTTP 401 error",
			domain: "ntfy.sh", 
			topic:  "test-topic",
			auth:   "",
			mockResponse: &http.Response{
				StatusCode: http.StatusUnauthorized,
				Body:       io.NopCloser(strings.NewReader("")),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedReq *http.Request
			
			mockClient := &MockHTTPClient{
				DoFunc: func(req *http.Request) (*http.Response, error) {
					capturedReq = req
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return tt.mockResponse, nil
				},
			}

			client := ntfy.NewClient(tt.domain, tt.topic, tt.auth, mockClient)
			reader, err := client.Connect()

			if (err != nil) != tt.wantErr {
				t.Errorf("Connect() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && reader == nil {
				t.Error("Expected reader but got nil")
			}

			if reader != nil {
				reader.Close()
			}

			if tt.validateReq && capturedReq != nil {
				if capturedReq.Method != http.MethodGet {
					t.Errorf("Expected GET request, got %s", capturedReq.Method)
				}
				
				expectedURL := "https://" + tt.domain + "/" + tt.topic + "/json"
				if capturedReq.URL.String() != expectedURL {
					t.Errorf("Expected URL %s, got %s", expectedURL, capturedReq.URL.String())
				}
				
				if tt.auth != "" {
					authHeader := capturedReq.Header.Get("Authorization")
					expectedAuth := "Bearer " + tt.auth
					if authHeader != expectedAuth {
						t.Errorf("Expected Authorization %s, got %s", expectedAuth, authHeader)
					}
				} else {
					authHeader := capturedReq.Header.Get("Authorization")
					if authHeader != "" {
						t.Errorf("Expected no Authorization header, got %s", authHeader)
					}
				}
			}
		})
	}
}

func TestNtfyHTTPClient_Connect_Integration(t *testing.T) {
	// Test with real HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		
		// Check URL path
		expectedPath := "/test-topic/json"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"event":"open","topic":"test-topic"}`))
	}))
	defer server.Close()

	// Create a mock HTTP client that calls our test server
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			// Replace the URL host with our test server
			req.URL.Host = strings.TrimPrefix(server.URL, "http://")
			req.URL.Scheme = "http"
			return http.DefaultClient.Do(req)
		},
	}
	
	// Use a valid domain for validation but the mock client will redirect to test server
	client := ntfy.NewClient("ntfy.sh", "test-topic", "", mockClient)
	
	reader, err := client.Connect()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	if reader == nil {
		t.Error("Expected reader but got nil")
	} else {
		reader.Close()
	}
}

func TestNtfyHTTPClient_Connect_URLEncoding(t *testing.T) {
	// Test URL encoding of topic names
	tests := []struct {
		name          string
		topic         string
		expectedPath  string
	}{
		{
			name:         "simple topic",
			topic:        "test-topic",
			expectedPath: "/test-topic/json",
		},
		{
			name:         "topic with special characters",
			topic:        "test_topic-123",
			expectedPath: "/test_topic-123/json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedReq *http.Request
			
			mockClient := &MockHTTPClient{
				DoFunc: func(req *http.Request) (*http.Response, error) {
					capturedReq = req
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader("")),
					}, nil
				},
			}

			client := ntfy.NewClient("ntfy.sh", tt.topic, "", mockClient)
			reader, err := client.Connect()
			
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if reader != nil {
				reader.Close()
			}
			
			if capturedReq != nil && capturedReq.URL.Path != tt.expectedPath {
				t.Errorf("Expected path %s, got %s", tt.expectedPath, capturedReq.URL.Path)
			}
		})
	}
}