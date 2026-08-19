package processor_test

// Split out of what was tests/unit/error_test.go and
// tests/unit/error_messaging_test.go, both of which mixed ntfy/slack/
// processor cases in one file back when tests lived in a flat external
// tree rather than beside the packages they cover. TestMessageProcessor_
// LogsCarryMessageID and TestMessageProcessor_FallbackDeliveryErrorIsNamed
// pin the operator-facing error text described in issue #16.

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/ozskywalker/ntfy-to-slack/internal/config"
	"github.com/ozskywalker/ntfy-to-slack/internal/processor"
	"github.com/ozskywalker/ntfy-to-slack/internal/testutil"
)

func TestMessageProcessor_ErrorRecovery(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		senderErrors   []error // Errors for each send attempt
		expectedSends  int     // How many sends should be attempted
		expectContinue bool    // Should processing continue after errors
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
			sender := &testutil.CustomErrorSender{
				SentMessages: []config.SlackMessage{},
				Errors:       tt.senderErrors,
				SendCount:    &sendCount,
			}

			p := processor.New(sender)
			reader := strings.NewReader(tt.input)

			err := p.ProcessStream(reader)

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

// Every per-message log line, and the terminal error, must carry the ntfy
// message id so a failure can be traced back to the message that caused it.
func TestMessageProcessor_LogsCarryMessageID(t *testing.T) {
	rec := captureRecords(t)

	p := processor.New(&testutil.MockMessageSender{SendError: errors.New("slack webhook returned 400: invalid_payload")})

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

	post := &testutil.MockPostProcessor{
		ProcessFunc: func(*config.NtfyMessage) (*config.SlackMessage, error) {
			return nil, errors.New("webhook returned status 502: upstream down")
		},
	}
	p := processor.NewWithPostProcessor(
		&testutil.MockMessageSender{SendError: errors.New("slack webhook returned 400: invalid_payload")},
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
