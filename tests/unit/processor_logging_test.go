package unit_test

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/ozskywalker/ntfy-to-slack/internal/config"
	"github.com/ozskywalker/ntfy-to-slack/internal/processor"
)

// captureHandler is a slog.Handler that records the messages logged so tests
// can assert on the pipeline-stage status messages (issue #17).
type captureHandler struct {
	messages []string
}

func (h *captureHandler) Enabled(context.Context, slog.Level) bool { return true }
func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	h.messages = append(h.messages, r.Message)
	return nil
}
func (h *captureHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *captureHandler) WithGroup(string) slog.Handler      { return h }

// captureMessages installs a capturing handler as the default logger and
// restores the original logger after the test.
func captureMessages(t *testing.T) *captureHandler {
	t.Helper()
	h := &captureHandler{}
	original := slog.Default()
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() { slog.SetDefault(original) })
	return h
}

func containsMessage(messages []string, want string) bool {
	for _, m := range messages {
		if m == want {
			return true
		}
	}
	return false
}

func TestMessageProcessor_LoggingStages(t *testing.T) {
	input := "{\"event\":\"message\",\"topic\":\"test\",\"title\":\"Alert\",\"message\":\"System down\"}"

	tests := []struct {
		name           string
		usePost        bool
		postProcErr    error
		expectMessages []string
	}{
		{
			name:    "passthrough without post-processor",
			usePost: false,
			expectMessages: []string{
				"received message from ntfy",
				"sending message to Slack",
			},
		},
		{
			name:    "successful post-processing",
			usePost: true,
			expectMessages: []string{
				"received message from ntfy",
				"sending message to post-processor",
				"post-processing complete, sending to Slack",
			},
		},
		{
			name:        "failed post-processing",
			usePost:     true,
			postProcErr: errors.New("webhook timed out"),
			expectMessages: []string{
				"received message from ntfy",
				"sending message to post-processor",
				"post-processing failed, using default format",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := captureMessages(t)
			sender := &MockMessageSender{}

			var p *processor.MessageProcessor
			if tt.usePost {
				post := &MockPostProcessor{
					ProcessFunc: func(msg *config.NtfyMessage) (*config.SlackMessage, error) {
						if tt.postProcErr != nil {
							return nil, tt.postProcErr
						}
						return &config.SlackMessage{Text: "Processed: " + msg.Message}, nil
					},
				}
				p = processor.NewWithPostProcessor(sender, post)
			} else {
				p = processor.New(sender)
			}

			if err := p.ProcessStream(strings.NewReader(input)); err != nil {
				t.Fatalf("ProcessStream error: %v", err)
			}

			for _, want := range tt.expectMessages {
				if !containsMessage(handler.messages, want) {
					t.Errorf("expected log message %q, captured %v", want, handler.messages)
				}
			}
		})
	}
}
