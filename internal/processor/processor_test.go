package processor

// This file is package processor, not processor_test: it exercises
// processMessage, handleMessageEvent, and createDefaultMessage directly,
// which are unexported and so unreachable from outside the package. Before
// tests were colocated with their source packages, several of these were
// dead skip stubs with no way to actually run them -- see this file's git
// history (tests/unit/processor_test.go and
// tests/unit/processor_postprocessor_test.go) for what they looked like
// before.

import (
	"errors"
	"strings"
	"testing"

	"github.com/ozskywalker/ntfy-to-slack/internal/config"
	"github.com/ozskywalker/ntfy-to-slack/internal/testutil"
)

func TestNewMessageProcessor(t *testing.T) {
	sender := &testutil.MockMessageSender{}
	p := New(sender)

	if p == nil {
		t.Error("New() returned nil")
	}
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
				{Text: "*Hello*: World"},
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
			sender := &testutil.MockMessageSender{
				SendError: tt.senderError,
			}
			p := New(sender)

			reader := strings.NewReader(tt.input)
			err := p.ProcessStream(reader)

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

func TestMessageProcessor_LastSeenID(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantID     string
		wantSeenOk bool
	}{
		{
			name:       "no messages seen yet",
			input:      "",
			wantID:     "",
			wantSeenOk: false,
		},
		{
			name: "tracks the id of the last well-formed line, including non-message events",
			input: `{"id":"open1","event":"open","topic":"test"}
{"id":"msg1","event":"message","topic":"test","message":"first"}
{"id":"keepalive1","event":"keepalive","topic":"test"}`,
			wantID:     "keepalive1",
			wantSeenOk: true,
		},
		{
			name:       "invalid JSON lines don't update the tracked id",
			input:      `{"id":"msg1","event":"message","topic":"test","message":"first"}` + "\n" + `{not valid json`,
			wantID:     "msg1",
			wantSeenOk: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := &testutil.MockMessageSender{}
			p := New(sender)

			if err := p.ProcessStream(strings.NewReader(tt.input)); err != nil {
				t.Fatalf("ProcessStream() error = %v", err)
			}

			id, ok := p.LastSeenID()
			if id != tt.wantID || ok != tt.wantSeenOk {
				t.Errorf("LastSeenID() = (%q, %v), want (%q, %v)", id, ok, tt.wantID, tt.wantSeenOk)
			}
		})
	}
}

func TestMessageProcessor_processMessage(t *testing.T) {
	tests := []struct {
		name        string
		msg         *config.NtfyMessage
		wantSent    bool
		wantErr     bool
		wantErrText string
	}{
		{
			name:     "open event is acknowledged, nothing sent",
			msg:      &config.NtfyMessage{Event: "open", Topic: "test"},
			wantSent: false,
		},
		{
			name:     "keepalive event is acknowledged, nothing sent",
			msg:      &config.NtfyMessage{Event: "keepalive", Topic: "test"},
			wantSent: false,
		},
		{
			name:     "message event is delivered",
			msg:      &config.NtfyMessage{Event: "message", Topic: "test", Title: "Alert", Message: "System down"},
			wantSent: true,
		},
		{
			name:     "unknown event is logged and ignored, nothing sent",
			msg:      &config.NtfyMessage{Event: "poll_request", Topic: "test"},
			wantSent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := &testutil.MockMessageSender{}
			p := New(sender)

			err := p.processMessage(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("processMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErrText != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErrText)) {
				t.Errorf("processMessage() error = %v, want it to contain %q", err, tt.wantErrText)
			}
			if got := len(sender.SentMessages) > 0; got != tt.wantSent {
				t.Errorf("message sent = %v, want %v (sent: %v)", got, tt.wantSent, sender.SentMessages)
			}
		})
	}
}

func TestMessageProcessor_handleMessageEvent(t *testing.T) {
	tests := []struct {
		name              string
		postProcessorFunc func(message *config.NtfyMessage) (*config.SlackMessage, error)
		inputMessage      *config.NtfyMessage
		expectedSlackText string
	}{
		{
			name: "successful post-processing",
			postProcessorFunc: func(message *config.NtfyMessage) (*config.SlackMessage, error) {
				return &config.SlackMessage{Text: "Custom: " + message.Title + " - " + message.Message}, nil
			},
			inputMessage: &config.NtfyMessage{
				Event:   "message",
				Title:   "Alert",
				Message: "System down",
			},
			expectedSlackText: "Custom: Alert - System down",
		},
		{
			name: "post-processor error falls back to default",
			postProcessorFunc: func(message *config.NtfyMessage) (*config.SlackMessage, error) {
				return nil, errors.New("processing failed")
			},
			inputMessage: &config.NtfyMessage{
				Event:   "message",
				Title:   "Alert",
				Message: "System down",
			},
			expectedSlackText: "*Alert*: System down",
		},
		{
			name: "message without title",
			postProcessorFunc: func(message *config.NtfyMessage) (*config.SlackMessage, error) {
				return &config.SlackMessage{Text: "Processed: " + message.Message}, nil
			},
			inputMessage: &config.NtfyMessage{
				Event:   "message",
				Title:   "",
				Message: "Simple message",
			},
			expectedSlackText: "Processed: Simple message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := &testutil.MockMessageSender{}
			post := &testutil.MockPostProcessor{ProcessFunc: tt.postProcessorFunc}
			p := NewWithPostProcessor(sender, post)

			if err := p.handleMessageEvent(tt.inputMessage); err != nil {
				t.Fatalf("handleMessageEvent() error = %v", err)
			}

			if post.CallCount != 1 {
				t.Errorf("expected post-processor to be called once, got %d", post.CallCount)
			}
			if len(sender.SentMessages) != 1 {
				t.Fatalf("expected 1 sent message, got %d", len(sender.SentMessages))
			}
			if sender.SentMessages[0].Text != tt.expectedSlackText {
				t.Errorf("expected text %q, got %q", tt.expectedSlackText, sender.SentMessages[0].Text)
			}
		})
	}
}

func TestMessageProcessor_handleMessageEvent_WithoutPostProcessor(t *testing.T) {
	sender := &testutil.MockMessageSender{}
	p := New(sender)

	msg := &config.NtfyMessage{Event: "message", Title: "Alert", Message: "System down"}
	if err := p.handleMessageEvent(msg); err != nil {
		t.Fatalf("handleMessageEvent() error = %v", err)
	}

	if len(sender.SentMessages) != 1 {
		t.Fatalf("expected 1 sent message, got %d", len(sender.SentMessages))
	}
	if want := "*Alert*: System down"; sender.SentMessages[0].Text != want {
		t.Errorf("expected text %q, got %q", want, sender.SentMessages[0].Text)
	}
}

func TestMessageProcessor_handleMessageEvent_FallbackDeliveryFails(t *testing.T) {
	sendErr := errors.New("slack webhook returned 400: invalid_payload")
	sender := &testutil.MockMessageSender{SendError: sendErr}
	post := &testutil.MockPostProcessor{
		ProcessFunc: func(*config.NtfyMessage) (*config.SlackMessage, error) {
			return nil, errors.New("webhook returned status 502: upstream down")
		},
	}
	p := NewWithPostProcessor(sender, post)

	err := p.handleMessageEvent(&config.NtfyMessage{Event: "message", Message: "test"})
	if err == nil {
		t.Fatal("expected an error when the fallback delivery also fails")
	}
	if !strings.Contains(err.Error(), "delivering default-format fallback") {
		t.Errorf("error %q should name the default-format fallback", err.Error())
	}
	if !strings.Contains(err.Error(), sendErr.Error()) {
		t.Errorf("error %q should preserve the underlying Slack reason", err.Error())
	}
}

func TestMessageProcessor_createDefaultMessage(t *testing.T) {
	tests := []struct {
		name         string
		message      *config.NtfyMessage
		expectedText string
	}{
		{
			name: "message with title",
			message: &config.NtfyMessage{
				Title:   "Alert",
				Message: "System issue",
			},
			expectedText: "*Alert*: System issue",
		},
		{
			name: "message without title",
			message: &config.NtfyMessage{
				Title:   "",
				Message: "Simple message",
			},
			expectedText: "Simple message",
		},
		{
			name: "message with empty title",
			message: &config.NtfyMessage{
				Title:   "",
				Message: "Another message",
			},
			expectedText: "Another message",
		},
	}

	p := New(&testutil.MockMessageSender{})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.createDefaultMessage(tt.message)

			if result == nil {
				t.Error("Expected result but got nil")
				return
			}

			if result.Text != tt.expectedText {
				t.Errorf("Expected text %q, got %q", tt.expectedText, result.Text)
			}
		})
	}
}
