package processor

import (
	"bufio"
	"encoding/json"
	"io"
	"log/slog"

	"github.com/ozskywalker/ntfy-to-slack/internal/config"
)

// MessageProcessor implements StreamProcessor interface
type MessageProcessor struct {
	sender        MessageSender
	postProcessor config.PostProcessor
}

// New creates a new message processor
func New(sender MessageSender) *MessageProcessor {
	return &MessageProcessor{
		sender: sender,
	}
}

// NewWithPostProcessor creates a new message processor with post-processing
func NewWithPostProcessor(sender MessageSender, postProcessor config.PostProcessor) *MessageProcessor {
	return &MessageProcessor{
		sender:        sender,
		postProcessor: postProcessor,
	}
}

// ProcessStream implements StreamProcessor interface
func (p *MessageProcessor) ProcessStream(reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		var msg config.NtfyMessage
		err := json.Unmarshal([]byte(scanner.Text()), &msg)
		if err != nil {
			slog.Error("error while processing ntfy message", "err", err, "text", scanner.Text())
			continue
		}
		
		if err := p.processMessage(&msg); err != nil {
			slog.Error("error processing message", "err", err)
		}
	}
	
	return scanner.Err()
}

// processMessage processes a single ntfy message
func (p *MessageProcessor) processMessage(msg *config.NtfyMessage) error {
	switch msg.Event {
	case "open":
		slog.Info("subscription established")
		return nil
	case "keepalive":
		slog.Debug("keepalive")
		return nil
	case "message":
		return p.handleMessageEvent(msg)
	default:
		slog.Warn("unknown message event", "event", msg.Event, "message", msg)
		return nil
	}
}

// handleMessageEvent handles ntfy message events
func (p *MessageProcessor) handleMessageEvent(msg *config.NtfyMessage) error {
	slog.Info("sending message", "title", msg.Title, "message", msg.Message)
	
	var slackMsg *config.SlackMessage
	var err error
	
	// Use post-processor if available
	if p.postProcessor != nil {
		slackMsg, err = p.postProcessor.Process(msg)
		if err != nil {
			slog.Warn("post-processing failed, using default format", "err", err)
			slackMsg = p.createDefaultMessage(msg)
		}
	} else {
		slackMsg = p.createDefaultMessage(msg)
	}
	
	return p.sender.Send(slackMsg)
}

// createDefaultMessage creates a default slack message format
func (p *MessageProcessor) createDefaultMessage(msg *config.NtfyMessage) *config.SlackMessage {
	if msg.Title != "" {
		return &config.SlackMessage{
			Text: "**" + msg.Title + "**: " + msg.Message,
		}
	}
	return &config.SlackMessage{
		Text: msg.Message,
	}
}