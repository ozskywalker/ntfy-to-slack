package unit_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/ozskywalker/ntfy-to-slack/internal/config"
)

func TestValidateDomain(t *testing.T) {
	tests := []struct {
		name     string
		domain   string
		wantErr  bool
		expected string
	}{
		{
			name:     "valid domain",
			domain:   "ntfy.sh",
			wantErr:  false,
			expected: "ntfy.sh",
		},
		{
			name:     "valid subdomain",
			domain:   "api.example.com",
			wantErr:  false,
			expected: "api.example.com",
		},
		{
			name:    "invalid domain - no tld",
			domain:  "localhost",
			wantErr: true,
		},
		{
			name:    "invalid domain - starts with dash",
			domain:  "-example.com",
			wantErr: true,
		},
		{
			name:    "invalid domain - ends with dash",
			domain:  "example-.com",
			wantErr: true,
		},
		{
			name:    "invalid domain - empty",
			domain:  "",
			wantErr: true,
		},
		{
			name:    "invalid domain - too long segment",
			domain:  "verylongdomainnamethatexceedsthemaximumlengthallowedfordomainsegmentslimit.com",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := config.ValidateDomain(tt.domain)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDomain() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result != tt.expected {
				t.Errorf("validateDomain() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestValidateTopic(t *testing.T) {
	tests := []struct {
		name     string
		topic    string
		wantErr  bool
		expected string
	}{
		{
			name:     "valid topic - alphanumeric",
			topic:    "mytopic123",
			wantErr:  false,
			expected: "mytopic123",
		},
		{
			name:     "valid topic - with dash",
			topic:    "my-topic",
			wantErr:  false,
			expected: "my-topic",
		},
		{
			name:     "valid topic - with underscore",
			topic:    "my_topic",
			wantErr:  false,
			expected: "my_topic",
		},
		{
			name:     "valid topic - single character",
			topic:    "a",
			wantErr:  false,
			expected: "a",
		},
		{
			name:     "valid topic - 64 characters",
			topic:    "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890ab",
			wantErr:  false,
			expected: "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890ab",
		},
		{
			name:    "invalid topic - empty",
			topic:   "",
			wantErr: true,
		},
		{
			name:    "invalid topic - with space",
			topic:   "my topic",
			wantErr: true,
		},
		{
			name:    "invalid topic - with special characters",
			topic:   "my@topic",
			wantErr: true,
		},
		{
			name:    "invalid topic - too long",
			topic:   "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890abc",
			wantErr: true,
		},
		{
			name:    "invalid topic - starts with dot",
			topic:   ".topic",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := config.ValidateTopic(tt.topic)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateTopic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result != tt.expected {
				t.Errorf("validateTopic() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNtfyMessageUnmarshaling(t *testing.T) {
	tests := []struct {
		name        string
		jsonInput   string
		expected    config.NtfyMessage
		shouldError bool
	}{
		{
			name:      "valid message event",
			jsonInput: `{"id":"abc123","time":1640995200,"event":"message","topic":"test","title":"Hello","message":"World"}`,
			expected: config.NtfyMessage{
				Id:      "abc123",
				Time:    1640995200,
				Event:   "message",
				Topic:   "test",
				Title:   "Hello",
				Message: "World",
			},
			shouldError: false,
		},
		{
			name:      "open event",
			jsonInput: `{"id":"","time":0,"event":"open","topic":"test","title":"","message":""}`,
			expected: config.NtfyMessage{
				Id:      "",
				Time:    0,
				Event:   "open",
				Topic:   "test",
				Title:   "",
				Message: "",
			},
			shouldError: false,
		},
		{
			name:        "invalid json",
			jsonInput:   `{"invalid json`,
			shouldError: true,
		},
		{
			name: "message with priority, tags, click, attachment and actions",
			jsonInput: `{
				"id":"abc123","time":1640995200,"expires":1640998800,
				"event":"message","topic":"test","title":"Hello","message":"World",
				"priority":5,
				"tags":["warning","skull"],
				"click":"https://example.com",
				"attachment":{"name":"file.jpg","url":"https://example.com/file.jpg","type":"image/jpeg","size":1234,"expires":1640998800},
				"actions":[{"id":"1","action":"view","label":"Open","url":"https://example.com","clear":true}]
			}`,
			expected: config.NtfyMessage{
				Id:       "abc123",
				Time:     1640995200,
				Expires:  1640998800,
				Event:    "message",
				Topic:    "test",
				Title:    "Hello",
				Message:  "World",
				Priority: 5,
				Tags:     []string{"warning", "skull"},
				Click:    "https://example.com",
				Attachment: &config.NtfyAttachment{
					Name:    "file.jpg",
					URL:     "https://example.com/file.jpg",
					Type:    "image/jpeg",
					Size:    1234,
					Expires: 1640998800,
				},
				Actions: []config.NtfyAction{
					{Id: "1", Action: "view", Label: "Open", URL: "https://example.com", Clear: true},
				},
			},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var msg config.NtfyMessage
			err := json.Unmarshal([]byte(tt.jsonInput), &msg)

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if !reflect.DeepEqual(msg, tt.expected) {
				t.Errorf("Expected %+v, got %+v", tt.expected, msg)
			}
		})
	}
}
