package slack

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/ozskywalker/ntfy-to-slack/internal/config"
)

// maxResponseBytes caps how much of the Slack response is read. Slack's
// replies are short status strings ("ok", "invalid_payload"), so this is
// generous while keeping a misbehaving endpoint from exhausting memory.
const maxResponseBytes = 1024

// Sender implements MessageSender interface
type Sender struct {
	webhookURL string
	client     config.HTTPClient
}

// NewSender creates a new Slack message sender
func NewSender(webhookURL string, client config.HTTPClient) *Sender {
	if client == nil {
		client = &http.Client{
			Timeout: 3 * time.Second,
		}
	}
	return &Sender{
		webhookURL: webhookURL,
		client:     client,
	}
}

// Send implements MessageSender interface
func (s *Sender) Send(message *config.SlackMessage) error {
	if message == nil {
		return errors.New("message is nil")
	}

	// Default to rendering Text as Slack mrkdwn unless an explicit value is
	// provided by the message (e.g. a webhook post-processor opting out).
	mrkdwn := true
	if message.Mrkdwn != nil {
		mrkdwn = *message.Mrkdwn
	}
	sendMsg := *message
	sendMsg.Mrkdwn = &mrkdwn

	jsonBytes, err := json.Marshal(sendMsg)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(
		http.MethodPost,
		s.webhookURL,
		bytes.NewBuffer(jsonBytes),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		if err := Body.Close(); err != nil {
			slog.Error("error closing response body", "err", err)
		}
	}(resp.Body)

	// Slack explains rejections in the response body (invalid_payload, no_text,
	// channel_not_found, ...), so keep it in scope for the error below instead
	// of discarding it after a debug log. The webhook URL is a credential and
	// must never appear in the returned error.
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return fmt.Errorf("reading Slack response (status %d): %w", resp.StatusCode, err)
	}
	slog.Debug("slack response", "status", resp.StatusCode, "body", string(body))

	if resp.StatusCode >= 400 {
		detail := strings.TrimSpace(string(body))
		if detail == "" {
			detail = "(empty response body)"
		}
		return fmt.Errorf("slack webhook returned %d: %s", resp.StatusCode, detail)
	}

	return nil
}
