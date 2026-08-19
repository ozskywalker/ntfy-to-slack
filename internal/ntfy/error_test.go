package ntfy_test

// Split out of what was tests/unit/error_test.go and
// tests/unit/error_messaging_test.go, both of which mixed ntfy/slack/
// processor cases in one file back when tests lived in a flat external
// tree rather than beside the packages they cover.

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ozskywalker/ntfy-to-slack/internal/ntfy"
	"github.com/ozskywalker/ntfy-to-slack/internal/testutil"
)

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
			name:    "empty domain after validation",
			domain:  "",
			topic:   "test",
			wantErr: true,
		},
		{
			name:    "empty topic after validation",
			domain:  "ntfy.sh",
			topic:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &testutil.MockHTTPClient{
				DoFunc: func(req *http.Request) (*http.Response, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return nil, errors.New("should not reach here")
				},
			}

			client := ntfy.NewClient(tt.domain, tt.topic, tt.auth, "", "", mockClient)
			_, err := client.Connect(context.Background(), "")

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

func TestNtfyClient_ConnectErrorIncludesResponseBody(t *testing.T) {
	client := &testutil.MockHTTPClient{
		DoFunc: func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusUnauthorized,
				Body:       io.NopCloser(strings.NewReader(`{"code":40101,"error":"unauthorized"}`)),
				Header:     make(http.Header),
			}, nil
		},
	}

	_, err := ntfy.NewClient("ntfy.sh", "test-topic", "", "", "", client).Connect(context.Background(), "")
	if err == nil {
		t.Fatal("expected an error for a 401 response")
	}
	for _, want := range []string{"invalid response code from ntfy: 401", "unauthorized"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q missing %q", err.Error(), want)
		}
	}
}
