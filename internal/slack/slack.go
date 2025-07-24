package slack

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/ozskywalker/ntfy-to-slack/internal/config"
)

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
	
	jsonBytes, err := json.Marshal(message)
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
	
	if body, err := io.ReadAll(resp.Body); err != nil {
		slog.Error("error parsing body", "err", err)
		return err
	} else {
		slog.Debug("slack response", "status", resp.StatusCode, "body", string(body))
	}
	
	if resp.StatusCode >= 400 {
		return errors.New("error status code " + strconv.FormatInt(int64(resp.StatusCode), 10))
	}
	
	return nil
}