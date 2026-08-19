package unit_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ozskywalker/ntfy-to-slack/internal/config"
	"github.com/ozskywalker/ntfy-to-slack/internal/slack"
)

// boolPtr returns a pointer to the given boolean value.
func boolPtr(v bool) *bool {
	return &v
}

func TestNewSlackSender(t *testing.T) {
	tests := []struct {
		name       string
		webhookURL string
		client     config.HTTPClient
		wantNil    bool
	}{
		{
			name:       "with custom client",
			webhookURL: "https://hooks.slack.com/test",
			client:     &MockHTTPClient{},
			wantNil:    false,
		},
		{
			name:       "with nil client creates default",
			webhookURL: "https://hooks.slack.com/test",
			client:     nil,
			wantNil:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := slack.NewSender(tt.webhookURL, tt.client)

			if (sender == nil) != tt.wantNil {
				t.Errorf("slack.NewSender() = %v, wantNil %v", sender == nil, tt.wantNil)
			}

			// Note: webhookURL is not exported, so we can't test it directly
			// The fact that NewSender() doesn't panic and returns a non-nil sender is sufficient
		})
	}
}

func TestSlackSender_Send(t *testing.T) {
	tests := []struct {
		name         string
		message      *config.SlackMessage
		mockResponse *http.Response
		mockError    error
		wantErr      bool
		validateReq  bool
		wantMrkdwn   *bool
	}{
		{
			name:    "successful send",
			message: &config.SlackMessage{Text: "test message"},
			mockResponse: &http.Response{
				StatusCode: http.StatusOK,
				Body:       http.NoBody,
			},
			wantErr:     false,
			validateReq: true,
			wantMrkdwn:  boolPtr(true),
		},
		{
			name:    "successful send with mrkdwn explicitly disabled",
			message: &config.SlackMessage{Text: "test message", Mrkdwn: boolPtr(false)},
			mockResponse: &http.Response{
				StatusCode: http.StatusOK,
				Body:       http.NoBody,
			},
			wantErr:     false,
			validateReq: true,
			wantMrkdwn:  boolPtr(false),
		},
		{
			name:    "nil message",
			message: nil,
			wantErr: true,
		},
		{
			name:      "HTTP client error",
			message:   &config.SlackMessage{Text: "test message"},
			mockError: errors.New("network error"),
			wantErr:   true,
		},
		{
			name:    "HTTP 400 error",
			message: &config.SlackMessage{Text: "test message"},
			mockResponse: &http.Response{
				StatusCode: http.StatusBadRequest,
				Body:       http.NoBody,
			},
			wantErr: true,
		},
		{
			name:    "HTTP 500 error",
			message: &config.SlackMessage{Text: "test message"},
			mockResponse: &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       http.NoBody,
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

			sender := slack.NewSender("https://hooks.slack.com/test", mockClient)
			err := sender.Send(tt.message)

			if (err != nil) != tt.wantErr {
				t.Errorf("Send() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.validateReq && capturedReq != nil {
				if capturedReq.Method != http.MethodPost {
					t.Errorf("Expected POST request, got %s", capturedReq.Method)
				}
				if capturedReq.Header.Get("Content-Type") != "application/json" {
					t.Errorf("Expected Content-Type application/json, got %s", capturedReq.Header.Get("Content-Type"))
				}
				if capturedReq.URL.String() != "https://hooks.slack.com/test" {
					t.Errorf("Expected URL https://hooks.slack.com/test, got %s", capturedReq.URL.String())
				}

				// Verify the JSON payload, including the mrkdwn field.
				body, err := io.ReadAll(capturedReq.Body)
				if err != nil {
					t.Errorf("Failed to read request body: %v", err)
				}
				var payload struct {
					Text   string `json:"text"`
					Mrkdwn *bool  `json:"mrkdwn"`
				}
				if err := json.Unmarshal(body, &payload); err != nil {
					t.Errorf("Failed to unmarshal request body %q: %v", string(body), err)
				}
				if payload.Text != tt.message.Text {
					t.Errorf("Expected payload text %q, got %q", tt.message.Text, payload.Text)
				}
				if tt.wantMrkdwn != nil {
					if payload.Mrkdwn == nil {
						t.Errorf("Expected mrkdwn %v in payload, got absent", *tt.wantMrkdwn)
					} else if *payload.Mrkdwn != *tt.wantMrkdwn {
						t.Errorf("Expected mrkdwn %v, got %v", *tt.wantMrkdwn, *payload.Mrkdwn)
					}
				}
			}
		})
	}
}

func TestSlackSender_Send_Integration(t *testing.T) {
	// Test with real HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	sender := slack.NewSender(server.URL, nil)
	err := sender.Send(&config.SlackMessage{Text: "integration test"})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestSlackSender_Send_Timeout(t *testing.T) {
	// Test timeout behavior
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second) // Longer than default timeout
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create sender with default client (has 3 second timeout)
	sender := slack.NewSender(server.URL, nil)
	err := sender.Send(&config.SlackMessage{Text: "timeout test"})

	if err == nil {
		t.Error("Expected timeout error but got none")
	}
}
