package unit_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ozskywalker/ntfy-to-slack/internal/config"
	"github.com/ozskywalker/ntfy-to-slack/internal/ntfy"
	"github.com/ozskywalker/ntfy-to-slack/internal/processor"
	"github.com/ozskywalker/ntfy-to-slack/internal/slack"
)

// These tests pin the operator-facing error text described in issue #16, where
// a Slack rejection surfaced only as "error status code 400" with no
// indication of which pipeline stage failed, which message it belonged to, or
// why Slack rejected it.

// recordedLog is a captured log record with its attributes flattened.
type recordedLog struct {
	message string
	attrs   map[string]string
}

// recorder collects records from a handler and every handler derived from it
// via WithAttrs, so attributes attached by slog.With are visible to assertions.
type recorder struct {
	records []recordedLog
}

type recordingHandler struct {
	rec  *recorder
	base []slog.Attr
}

func (h *recordingHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *recordingHandler) Handle(_ context.Context, r slog.Record) error {
	entry := recordedLog{message: r.Message, attrs: map[string]string{}}
	for _, a := range h.base {
		entry.attrs[a.Key] = a.Value.String()
	}
	r.Attrs(func(a slog.Attr) bool {
		entry.attrs[a.Key] = a.Value.String()
		return true
	})
	h.rec.records = append(h.rec.records, entry)
	return nil
}

func (h *recordingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &recordingHandler{rec: h.rec, base: append(append([]slog.Attr{}, h.base...), attrs...)}
}

func (h *recordingHandler) WithGroup(string) slog.Handler { return h }

// captureRecords installs an attribute-capturing handler as the default logger
// and restores the original logger afterwards.
func captureRecords(t *testing.T) *recorder {
	t.Helper()
	rec := &recorder{}
	original := slog.Default()
	slog.SetDefault(slog.New(&recordingHandler{rec: rec}))
	t.Cleanup(func() { slog.SetDefault(original) })
	return rec
}

func (r *recorder) find(message string) (recordedLog, bool) {
	for _, entry := range r.records {
		if entry.message == message {
			return entry, true
		}
	}
	return recordedLog{}, false
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
	client := &MockHTTPClient{
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

// Every per-message log line, and the terminal error, must carry the ntfy
// message id so a failure can be traced back to the message that caused it.
func TestMessageProcessor_LogsCarryMessageID(t *testing.T) {
	rec := captureRecords(t)

	p := processor.New(&MockMessageSender{SendError: errors.New("slack webhook returned 400: invalid_payload")})

	input := `{"id":"Bx9kL2mQ","event":"message","topic":"alerts","title":"","message":"blah test"}`
	if err := p.ProcessStream(strings.NewReader(input)); err != nil {
		t.Fatalf("ProcessStream error: %v", err)
	}

	for _, message := range []string{
		"received message from ntfy",
		"sending message to Slack",
		"error processing message",
	} {
		entry, ok := rec.find(message)
		if !ok {
			t.Fatalf("expected log record %q, captured %v", message, rec.records)
		}
		if entry.attrs["msg_id"] != "Bx9kL2mQ" {
			t.Errorf("record %q: msg_id = %q, want %q", message, entry.attrs["msg_id"], "Bx9kL2mQ")
		}
		if entry.attrs["topic"] != "alerts" {
			t.Errorf("record %q: topic = %q, want %q", message, entry.attrs["topic"], "alerts")
		}
	}
}

// When post-processing fails the message is still delivered, but in the default
// format. If that delivery also fails, the error must say which one it was.
func TestMessageProcessor_FallbackDeliveryErrorIsNamed(t *testing.T) {
	rec := captureRecords(t)

	post := &MockPostProcessor{
		ProcessFunc: func(*config.NtfyMessage) (*config.SlackMessage, error) {
			return nil, errors.New("webhook returned status 502: upstream down")
		},
	}
	p := processor.NewWithPostProcessor(
		&MockMessageSender{SendError: errors.New("slack webhook returned 400: invalid_payload")},
		post,
	)

	input := `{"id":"Bx9kL2mQ","event":"message","topic":"alerts","message":"blah test"}`
	if err := p.ProcessStream(strings.NewReader(input)); err != nil {
		t.Fatalf("ProcessStream error: %v", err)
	}

	entry, ok := rec.find("error processing message")
	if !ok {
		t.Fatalf("expected an error record for the failed fallback delivery, captured %v", rec.records)
	}
	if !strings.Contains(entry.attrs["err"], "delivering default-format fallback") {
		t.Errorf("err = %q, want it to name the default-format fallback", entry.attrs["err"])
	}
	if !strings.Contains(entry.attrs["err"], "invalid_payload") {
		t.Errorf("err = %q, want it to preserve the underlying Slack reason", entry.attrs["err"])
	}
}

func TestWebhookPostProcessor_ErrorNamesTargetAndAttempts(t *testing.T) {
	t.Run("client error names the target", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = w.Write([]byte(`{"error":"template var 'severity' missing"}`))
		}))
		defer server.Close()

		_, err := config.NewWebhookPostProcessorWithConfig(server.URL, 5, 0, 1).
			Process(&config.NtfyMessage{Message: "hi"})
		if err == nil {
			t.Fatal("expected an error for a 422 response")
		}
		for _, want := range []string{"webhook post-processing failed", server.URL, "status 422", "severity"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("error %q missing %q", err.Error(), want)
			}
		}
	})

	t.Run("exhausted retries report attempts, not retries", func(t *testing.T) {
		var calls int
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			calls++
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte("upstream down"))
		}))
		defer server.Close()

		// maxRetries=1 means 2 attempts: the original plus one retry.
		_, err := config.NewWebhookPostProcessorWithConfig(server.URL, 5, 1, 1).
			Process(&config.NtfyMessage{Message: "hi"})
		if err == nil {
			t.Fatal("expected an error after retries are exhausted")
		}
		if calls != 2 {
			t.Errorf("server saw %d calls, want 2", calls)
		}
		for _, want := range []string{"failed after 2 attempts", server.URL, "status 502", "upstream down"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("error %q missing %q", err.Error(), want)
			}
		}
	})
}

func TestNtfyClient_ConnectErrorIncludesResponseBody(t *testing.T) {
	client := &MockHTTPClient{
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
