package processor

import (
	"bufio"
	"encoding/json"
	"fmt"
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

// maxLineBytes bounds how large a single ntfy JSON line (message text plus
// attachment metadata) may be before ProcessStream gives up on the
// connection. bufio.Scanner's default 64KB limit is too easy to exceed with
// a large message or attachment, which would otherwise surface as a fatal
// ErrTooLong on an otherwise-healthy stream.
const maxLineBytes = 1024 * 1024

// ProcessStream implements StreamProcessor interface
func (p *MessageProcessor) ProcessStream(reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), maxLineBytes)
	for scanner.Scan() {
		var msg config.NtfyMessage
		err := json.Unmarshal([]byte(scanner.Text()), &msg)
		if err != nil {
			slog.Error("error while processing ntfy message", "err", err, "text", scanner.Text())
			continue
		}

		if err := p.processMessage(&msg); err != nil {
			slog.Error("error processing message", "err", err, "msg_id", msg.Id, "topic", msg.Topic)
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
	// Scope every line for this message to the id ntfy assigned it, so a
	// failure can be correlated back to the message that produced it.
	log := slog.With("msg_id", msg.Id, "topic", msg.Topic)
	log.Info("received message from ntfy", "title", msg.Title, "message", msg.Message)

	// Use post-processor if available
	if p.postProcessor != nil {
		log.Info("sending message to post-processor")
		slackMsg, err := p.postProcessor.Process(msg)
		if err != nil {
			log.Warn("post-processing failed, using default format", "err", err, "title", msg.Title)
			if sendErr := p.sender.Send(p.createDefaultMessage(msg)); sendErr != nil {
				// Name the fallback explicitly: what reached Slack was the
				// default format, not the post-processed message.
				return fmt.Errorf("delivering default-format fallback: %w", sendErr)
			}
			return nil
		}
		log.Info("post-processing complete, sending to Slack")
		log.Debug("post-processed message", "text", slackMsg.Text)
		return p.sender.Send(slackMsg)
	}

	log.Info("sending message to Slack", "title", msg.Title)
	return p.sender.Send(p.createDefaultMessage(msg))
}

// createDefaultMessage creates a default slack message format
func (p *MessageProcessor) createDefaultMessage(msg *config.NtfyMessage) *config.SlackMessage {
	if msg.Title != "" {
		return &config.SlackMessage{
			Text: "*" + msg.Title + "*: " + msg.Message,
		}
	}
	return &config.SlackMessage{
		Text: msg.Message,
	}
}
