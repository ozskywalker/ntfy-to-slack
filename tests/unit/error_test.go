package unit_test

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ozskywalker/ntfy-to-slack/internal/config"
	"github.com/ozskywalker/ntfy-to-slack/internal/ntfy"
	"github.com/ozskywalker/ntfy-to-slack/internal/processor"
	"github.com/ozskywalker/ntfy-to-slack/internal/slack"
)


// Test various error conditions and edge cases across the application

func TestNtfyClient_ErrorConditions(t *testing.T) {
	tests := []struct {
		name        string
		domain      string
		topic       string
		auth        string
		mockError   error
		wantErr     bool
		errContains string
	}{
		{
			name:        "network timeout",
			domain:      "ntfy.sh",
			topic:       "test",
			mockError:   errors.New("context deadline exceeded"),
			wantErr:     true,
			errContains: "context deadline exceeded",
		},
		{
			name:        "DNS resolution failure",
			domain:      "ntfy.sh",
			topic:       "test", 
			mockError:   errors.New("no such host"),
			wantErr:     true,
			errContains: "no such host",
		},
		{
			name:        "connection refused",
			domain:      "ntfy.sh",
			topic:       "test",
			mockError:   errors.New("connection refused"),
			wantErr:     true,
			errContains: "connection refused",
		},
		{
			name:      "empty domain after validation",
			domain:    "",
			topic:     "test",
			wantErr:   true,
		},
		{
			name:      "empty topic after validation", 
			domain:    "ntfy.sh",
			topic:     "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockHTTPClient{
				DoFunc: func(req *http.Request) (*http.Response, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return nil, errors.New("should not reach here")
				},
			}

			client := ntfy.NewClient(tt.domain, tt.topic, tt.auth, mockClient)
			_, err := client.Connect()

			if (err != nil) != tt.wantErr {
				t.Errorf("Connect() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.errContains != "" && err != nil {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error to contain %q, got: %v", tt.errContains, err)
				}
			}
		})
	}
}

func TestSlackSender_ErrorConditions(t *testing.T) {
	tests := []struct {
		name        string
		message     *config.SlackMessage
		mockError   error
		wantErr     bool
		errContains string
	}{
		{
			name:        "network timeout",
			message:     &config.SlackMessage{Text: "test"},
			mockError:   errors.New("context deadline exceeded"),
			wantErr:     true,
			errContains: "context deadline exceeded",
		},
		{
			name:        "DNS failure",
			message:     &config.SlackMessage{Text: "test"},
			mockError:   errors.New("no such host"),
			wantErr:     true,
			errContains: "no such host",
		},
		{
			name:        "connection refused",
			message:     &config.SlackMessage{Text: "test"},
			mockError:   errors.New("connection refused"),
			wantErr:     true,
			errContains: "connection refused",
		},
		{
			name:    "nil message",
			message: nil,
			wantErr: true,
		},
		{
			name:    "empty message text",
			message: &config.SlackMessage{Text: ""},
			wantErr: false, // Empty text is valid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockHTTPClient{
				DoFunc: func(req *http.Request) (*http.Response, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					// Return success response for cases without mockError
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader("ok")),
					}, nil
				},
			}

			sender := slack.NewSender("https://hooks.slack.com/test", mockClient)
			err := sender.Send(tt.message)

			if (err != nil) != tt.wantErr {
				t.Errorf("Send() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.errContains != "" && err != nil {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error to contain %q, got: %v", tt.errContains, err)
				}
			}
		})
	}
}

func TestMessageProcessor_ErrorRecovery(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		senderErrors    []error // Errors for each send attempt
		expectedSends   int     // How many sends should be attempted
		expectContinue  bool    // Should processing continue after errors
	}{
		{
			name: "continue processing after send error",
			input: `{"event":"message","title":"First","message":"Message 1"}
{"event":"message","title":"Second","message":"Message 2"}`,
			senderErrors:   []error{errors.New("first send failed"), nil},
			expectedSends:  2,
			expectContinue: true,
		},
		{
			name: "continue processing after multiple send errors",
			input: `{"event":"message","title":"First","message":"Message 1"}
{"event":"message","title":"Second","message":"Message 2"}
{"event":"message","title":"Third","message":"Message 3"}`,
			senderErrors:   []error{errors.New("failed"), errors.New("failed"), nil},
			expectedSends:  3,
			expectContinue: true,
		},
		{
			name: "mixed invalid JSON and valid messages",
			input: `invalid json line
{"event":"message","title":"Valid","message":"Message"}
another invalid line
{"event":"message","title":"Another","message":"Valid"}`,
			senderErrors:   []error{nil, nil},
			expectedSends:  2,
			expectContinue: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sendCount := 0
			
			// Create a custom sender that tracks calls and simulates errors
			sender := &CustomErrorSender{
				SentMessages: []config.SlackMessage{},
				Errors:       tt.senderErrors,
				SendCount:    &sendCount,
			}

			processor := processor.New(sender)
			reader := strings.NewReader(tt.input)
			
			err := processor.ProcessStream(reader)
			
			// ProcessStream should not return errors for individual message failures
			if err != nil {
				t.Errorf("ProcessStream() should not return error for individual failures, got: %v", err)
			}

			if sendCount != tt.expectedSends {
				t.Errorf("Expected %d send attempts, got %d", tt.expectedSends, sendCount)
			}
		})
	}
}


func TestValidation_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		domain  string
		topic   string
		wantErr bool
	}{
		{
			name:    "domain with single character",
			domain:  "a.com",
			topic:   "test",
			wantErr: false,
		},
		{
			name:    "domain with maximum valid length",
			domain:  strings.Repeat("a", 63) + ".com",
			topic:   "test",
			wantErr: false,
		},
		{
			name:    "topic with single character",
			domain:  "ntfy.sh",
			topic:   "a",
			wantErr: false,
		},
		{
			name:    "topic with special allowed characters",
			domain:  "ntfy.sh",
			topic:   "test_topic-123",
			wantErr: false,
		},
		{
			name:    "domain with consecutive dots",
			domain:  "test..example.com",
			topic:   "test",
			wantErr: true,
		},
		{
			name:    "topic with consecutive underscores",
			domain:  "ntfy.sh",
			topic:   "test__topic",
			wantErr: false, // This should be valid
		},
		{
			name:    "topic with consecutive dashes",
			domain:  "ntfy.sh",
			topic:   "test--topic",
			wantErr: false, // This should be valid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, domainErr := config.ValidateDomain(tt.domain)
			_, topicErr := config.ValidateTopic(tt.topic)
			
			hasErr := domainErr != nil || topicErr != nil
			if hasErr != tt.wantErr {
				t.Errorf("validation error = %v (domain) or %v (topic), wantErr %v", domainErr, topicErr, tt.wantErr)
			}
		})
	}
}