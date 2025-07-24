package unit_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/ozskywalker/ntfy-to-slack/internal/config"
	"github.com/ozskywalker/ntfy-to-slack/internal/processor"
)

func TestNewMessageProcessor(t *testing.T) {
	sender := &MockMessageSender{}
	processor := processor.New(sender)

	if processor == nil {
		t.Error("processor.New() returned nil")
	}

	// Note: internal fields are not exported, so we can't test them directly
	// The fact that processor.New() doesn't panic and returns a non-nil processor is sufficient
}

func TestMessageProcessor_ProcessStream(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		expectedMessages []config.SlackMessage
		senderError      error
		wantErr          bool
	}{
		{
			name: "process multiple message types",
			input: `{"event":"open","topic":"test"}
{"id":"msg1","time":1640995200,"event":"message","topic":"test","title":"Hello","message":"World"}
{"event":"keepalive","topic":"test"}
{"id":"msg2","time":1640995201,"event":"message","topic":"test","title":"","message":"Simple message"}`,
			expectedMessages: []config.SlackMessage{
				{Text: "**Hello**: World"},
				{Text: "Simple message"},
			},
			wantErr: false,
		},
		{
			name:             "process only control messages",
			input:            `{"event":"open","topic":"test"}` + "\n" + `{"event":"keepalive","topic":"test"}`,
			expectedMessages: []config.SlackMessage{},
			wantErr:          false,
		},
		{
			name:             "handle invalid JSON",
			input:            `{"invalid json` + "\n" + `{"event":"open","topic":"test"}`,
			expectedMessages: []config.SlackMessage{},
			wantErr:          false,
		},
		{
			name: "handle unknown event types",
			input: `{"event":"unknown","topic":"test"}
{"id":"msg1","time":1640995200,"event":"message","topic":"test","title":"","message":"Test"}`,
			expectedMessages: []config.SlackMessage{
				{Text: "Test"},
			},
			wantErr: false,
		},
		{
			name:        "sender error propagated",
			input:       `{"id":"msg1","time":1640995200,"event":"message","topic":"test","title":"","message":"Test"}`,
			senderError: errors.New("sender error"),
			wantErr:     false, // ProcessStream doesn't return sender errors, just logs them
		},
		{
			name:             "empty input",
			input:            "",
			expectedMessages: []config.SlackMessage{},
			wantErr:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := &MockMessageSender{
				SendError: tt.senderError,
			}
			processor := processor.New(sender)

			reader := strings.NewReader(tt.input)
			err := processor.ProcessStream(reader)

			if (err != nil) != tt.wantErr {
				t.Errorf("ProcessStream() error = %v, wantErr %v", err, tt.wantErr)
			}

			if len(sender.SentMessages) != len(tt.expectedMessages) {
				t.Errorf("Expected %d messages, got %d", len(tt.expectedMessages), len(sender.SentMessages))
			}

			for i, expected := range tt.expectedMessages {
				if i >= len(sender.SentMessages) {
					t.Errorf("Missing expected message: %v", expected)
					continue
				}
				if sender.SentMessages[i].Text != expected.Text {
					t.Errorf("Message %d: expected %q, got %q", i, expected.Text, sender.SentMessages[i].Text)
				}
			}
		})
	}
}

func TestMessageProcessor_processMessage(t *testing.T) {
	// processMessage is not exported, skip this test as it tests internal implementation
	t.Skip("processMessage is not exported - test internal implementation")
}

func TestMessageProcessor_handleMessageEvent(t *testing.T) {
	// handleMessageEvent is not exported, skip this test as it tests internal implementation
	t.Skip("handleMessageEvent is not exported - test internal implementation")
}
