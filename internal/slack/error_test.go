package slack_test

// Split out of what was tests/unit/error_test.go and
// tests/unit/error_messaging_test.go, both of which mixed ntfy/slack/
// processor cases in one file back when tests lived in a flat external
// tree rather than beside the packages they cover. These pin the
// operator-facing error text described in issue #16, where a Slack
// rejection surfaced only as "error status code 400" with no indication of
// which pipeline stage failed, which message it belonged to, or why Slack
// rejected it.

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ozskywalker/ntfy-to-slack/internal/config"
	"github.com/ozskywalker/ntfy-to-slack/internal/slack"
	"github.com/ozskywalker/ntfy-to-slack/internal/testutil"
)

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
			mockClient := &testutil.MockHTTPClient{
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

// failingBody is a ReadCloser that always fails to read.
type failingBody struct{}

func (failingBody) Read([]byte) (int, error) { return 0, errors.New("connection reset") }
func (failingBody) Close() error             { return nil }

func TestSlackSender_ErrorIncludesResponseBody(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantParts  []string
	}{
		{
			name:       "400 invalid_payload",
			statusCode: http.StatusBadRequest,
			body:       "invalid_payload",
			wantParts:  []string{"slack webhook returned 400", "invalid_payload"},
		},
		{
			name:       "403 invalid_token",
			statusCode: http.StatusForbidden,
			body:       "invalid_token",
			wantParts:  []string{"slack webhook returned 403", "invalid_token"},
		},
		{
			name:       "500 with empty body",
			statusCode: http.StatusInternalServerError,
			body:       "",
			wantParts:  []string{"slack webhook returned 500", "(empty response body)"},
		},
		{
			name:       "body is trimmed",
			statusCode: http.StatusBadRequest,
			body:       "  no_text\n",
			wantParts:  []string{"slack webhook returned 400: no_text"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			err := slack.NewSender(server.URL, nil).Send(&config.SlackMessage{Text: "hello"})
			if err == nil {
				t.Fatal("expected an error for a >=400 response")
			}
			for _, want := range tt.wantParts {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q missing %q", err.Error(), want)
				}
			}
		})
	}
}

// The Slack webhook URL is a bearer credential and must never be echoed into
// an error that lands in logs.
func TestSlackSender_ErrorDoesNotLeakWebhookURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("invalid_payload"))
	}))
	defer server.Close()

	const secret = "XXXXXXXXXXXXXXXXXXXXXXXX"
	err := slack.NewSender(server.URL+"/services/T00000000/B00000000/"+secret, nil).
		Send(&config.SlackMessage{Text: "hello"})
	if err == nil {
		t.Fatal("expected an error for a 400 response")
	}
	if strings.Contains(err.Error(), secret) {
		t.Errorf("error leaks the Slack webhook URL: %q", err.Error())
	}
}

// A response whose body cannot be read should still report the status it was
// reading, rather than a bare I/O error.
func TestSlackSender_ReadErrorReportsStatus(t *testing.T) {
	client := &testutil.MockHTTPClient{
		DoFunc: func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       failingBody{},
				Header:     make(http.Header),
			}, nil
		},
	}

	err := slack.NewSender("https://hooks.slack.com/services/test", client).
		Send(&config.SlackMessage{Text: "hello"})
	if err == nil {
		t.Fatal("expected an error when the response body cannot be read")
	}
	if !strings.Contains(err.Error(), "status 200") {
		t.Errorf("error %q should name the status it was reading", err.Error())
	}
}
