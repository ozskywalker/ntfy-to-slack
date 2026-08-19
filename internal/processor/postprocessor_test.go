package processor_test

import (
	"strings"
	"testing"

	"github.com/ozskywalker/ntfy-to-slack/internal/config"
	"github.com/ozskywalker/ntfy-to-slack/internal/processor"
	"github.com/ozskywalker/ntfy-to-slack/internal/testutil"
)

func TestNewMessageProcessorWithPostProcessor(t *testing.T) {
	sender := &testutil.MockMessageSender{}
	postProcessor := &testutil.MockPostProcessor{}

	p := processor.NewWithPostProcessor(sender, postProcessor)

	if p == nil {
		t.Error("Expected processor but got nil")
	}
}

func TestMessageProcessor_ProcessStream_WithPostProcessor(t *testing.T) {
	input := `{"event":"open","topic":"test"}
{"id":"msg1","time":1640995200,"event":"message","topic":"test","title":"Alert","message":"System issue"}
{"event":"keepalive","topic":"test"}
{"id":"msg2","time":1640995201,"event":"message","topic":"test","title":"","message":"Simple message"}`

	sender := &testutil.MockMessageSender{}
	postProcessor := &testutil.MockPostProcessor{
		ProcessFunc: func(message *config.NtfyMessage) (*config.SlackMessage, error) {
			return &config.SlackMessage{Text: "🔔 " + message.Title + ": " + message.Message}, nil
		},
	}

	p := processor.NewWithPostProcessor(sender, postProcessor)
	reader := strings.NewReader(input)

	err := p.ProcessStream(reader)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Should have processed 2 message events
	if postProcessor.CallCount != 2 {
		t.Errorf("Expected post-processor to be called 2 times, got %d", postProcessor.CallCount)
	}

	if len(sender.SentMessages) != 2 {
		t.Errorf("Expected 2 sent messages, got %d", len(sender.SentMessages))
		return
	}

	expectedMessages := []string{
		"🔔 Alert: System issue",
		"🔔 : Simple message",
	}

	for i, expected := range expectedMessages {
		if sender.SentMessages[i].Text != expected {
			t.Errorf("Message %d: expected %q, got %q", i, expected, sender.SentMessages[i].Text)
		}
	}
}
