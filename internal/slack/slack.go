package slack

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ozskywalker/ntfy-to-slack/internal/config"
)

// maxResponseBytes caps how much of the Slack response is read. Slack's
// replies are short status strings ("ok", "invalid_payload"), so this is
// generous while keeping a misbehaving endpoint from exhausting memory.
const maxResponseBytes = 1024

// maxSendAttempts bounds retries for a transient Slack failure (429 or 5xx).
// Kept small and non-configurable: unlike the webhook post-processor, Send
// runs synchronously in the single message-processing loop, so retrying
// here delays every subsequent ntfy message, not just this one.
const maxSendAttempts = 3

// baseBackoff/maxBackoff bound the delay between retries when Slack's
// response doesn't include a Retry-After header.
const (
	baseBackoff = 1 * time.Second
	maxBackoff  = 10 * time.Second
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

// Send implements MessageSender interface. A 429 or 5xx response from Slack
// is retried (honoring a Retry-After header when Slack sends one); a
// network-level failure or any other 4xx is returned immediately.
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

	var lastErr error
	var retryAfter time.Duration
	for attempt := 0; attempt < maxSendAttempts; attempt++ {
		if attempt > 0 {
			delay := retryAfter
			if delay <= 0 {
				delay = backoffDelay(attempt)
			}
			slog.Debug("retrying slack send", "attempt", attempt+1, "delay", delay)
			time.Sleep(delay)
		}

		var retryable bool
		retryable, retryAfter, lastErr = s.sendOnce(jsonBytes)
		if lastErr == nil {
			return nil
		}
		if !retryable {
			return lastErr
		}
	}

	return fmt.Errorf("slack send failed after %d attempts: %w", maxSendAttempts, lastErr)
}

// sendOnce makes a single delivery attempt and reports whether the failure
// (if any) is worth retrying, and how long Slack asked callers to wait
// before retrying (via Retry-After), if it said anything.
func (s *Sender) sendOnce(jsonBytes []byte) (retryable bool, retryAfter time.Duration, err error) {
	req, err := http.NewRequest(
		http.MethodPost,
		s.webhookURL,
		bytes.NewBuffer(jsonBytes),
	)
	if err != nil {
		return false, 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return false, 0, err
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
		return false, 0, fmt.Errorf("reading Slack response (status %d): %w", resp.StatusCode, err)
	}
	slog.Debug("slack response", "status", resp.StatusCode, "body", string(body))

	if resp.StatusCode < 400 {
		return false, 0, nil
	}

	detail := strings.TrimSpace(string(body))
	if detail == "" {
		detail = "(empty response body)"
	}
	sendErr := fmt.Errorf("slack webhook returned %d: %s", resp.StatusCode, detail)

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return true, parseRetryAfter(resp.Header.Get("Retry-After")), sendErr
	}
	return false, 0, sendErr
}

// backoffDelay returns the delay before the given retry attempt (1-indexed:
// attempt 1 is the first retry), doubling each time and capped at
// maxBackoff.
func backoffDelay(attempt int) time.Duration {
	delay := baseBackoff << (attempt - 1)
	if delay > maxBackoff {
		return maxBackoff
	}
	return delay
}

// parseRetryAfter reads a Retry-After header value expressed in seconds (the
// only form Slack sends). It returns 0 if the header is absent or
// unparseable, signaling the caller should fall back to its own backoff.
func parseRetryAfter(v string) time.Duration {
	if v == "" {
		return 0
	}
	secs, err := strconv.Atoi(v)
	if err != nil || secs <= 0 {
		return 0
	}
	return time.Duration(secs) * time.Second
}
