package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/ozskywalker/ntfy-to-slack/internal/version"
)

// HTTPClient interface for HTTP operations
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// NtfyMessage represents a message from ntfy
type NtfyMessage struct {
	Id      string
	Time    int64
	Event   string
	Topic   string
	Title   string
	Message string
}

// SlackMessage represents a message to send to Slack
type SlackMessage struct {
	Text string `json:"text"`
	// Mrkdwn controls whether the Text field is rendered as Slack mrkdwn.
	// A nil value (absent from JSON) lets the Slack sender apply its default
	// of enabling mrkdwn, while an explicit false opts out.
	Mrkdwn *bool `json:"mrkdwn,omitempty"`
}

// PostProcessor defines the interface for message post-processing
type PostProcessor interface {
	Process(message *NtfyMessage) (*SlackMessage, error)
}

// TemplatePostProcessor processes messages using Go text/template templates
type TemplatePostProcessor struct {
	template *template.Template
}

// NewTemplatePostProcessor creates a new Go text/template post-processor
func NewTemplatePostProcessor(templateContent string) (*TemplatePostProcessor, error) {
	// Validate template syntax by parsing it
	tmpl, err := template.New("message").Parse(templateContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	// Additional validation: try executing with empty data to catch runtime issues
	if err := validateTemplateExecution(tmpl); err != nil {
		return nil, fmt.Errorf("template validation failed: %w", err)
	}

	return &TemplatePostProcessor{
		template: tmpl,
	}, nil
}

// validateTemplateExecution tests template execution with sample data
func validateTemplateExecution(tmpl *template.Template) error {
	// Test with empty NtfyMessage
	testMessage := &NtfyMessage{}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, testMessage); err != nil {
		return fmt.Errorf("template execution failed with empty data: %w", err)
	}

	// Test with sample data to catch field access issues
	sampleMessage := &NtfyMessage{
		Id:      "test-id",
		Time:    1640995200,
		Event:   "message",
		Topic:   "test-topic",
		Title:   "Test Title",
		Message: "Test Message",
	}

	buf.Reset()
	if err := tmpl.Execute(&buf, sampleMessage); err != nil {
		return fmt.Errorf("template execution failed with sample data: %w", err)
	}

	return nil
}

// NewTemplatePostProcessorFromFile creates a new Go text/template post-processor from file
func NewTemplatePostProcessorFromFile(filePath string) (*TemplatePostProcessor, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read template file %s: %w", filePath, err)
	}

	return NewTemplatePostProcessor(string(content))
}

// Process processes the ntfy message using the Go text/template
func (m *TemplatePostProcessor) Process(message *NtfyMessage) (*SlackMessage, error) {
	var buf bytes.Buffer
	err := m.template.Execute(&buf, message)
	if err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	// Slack rejects a message with no text (the "no_text" error), so treat
	// blank output the same as any other post-processing failure: return an
	// error and let the caller fall back to default formatting instead of
	// silently sending something Slack will reject.
	if strings.TrimSpace(buf.String()) == "" {
		return nil, fmt.Errorf("template produced empty output")
	}

	return &SlackMessage{
		Text: buf.String(),
	}, nil
}

// WebhookPostProcessor processes messages by calling an external webhook
type WebhookPostProcessor struct {
	webhookURL        string
	client            HTTPClient
	timeoutSeconds    int
	maxRetries        int
	maxResponseSizeMB int
}

// NewWebhookPostProcessor creates a new webhook post-processor
func NewWebhookPostProcessor(webhookURL string, client HTTPClient) *WebhookPostProcessor {
	if client == nil {
		client = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	return &WebhookPostProcessor{
		webhookURL:        webhookURL,
		client:            client,
		timeoutSeconds:    30,
		maxRetries:        3,
		maxResponseSizeMB: 1,
	}
}

// NewWebhookPostProcessorWithConfig creates a new webhook post-processor with configuration
func NewWebhookPostProcessorWithConfig(webhookURL string, timeoutSeconds, maxRetries, maxResponseSizeMB int) *WebhookPostProcessor {
	client := &http.Client{
		Timeout: time.Duration(timeoutSeconds) * time.Second,
	}

	return &WebhookPostProcessor{
		webhookURL:        webhookURL,
		client:            client,
		timeoutSeconds:    timeoutSeconds,
		maxRetries:        maxRetries,
		maxResponseSizeMB: maxResponseSizeMB,
	}
}

// Process processes the ntfy message by calling the webhook with retry logic
func (w *WebhookPostProcessor) Process(message *NtfyMessage) (*SlackMessage, error) {
	// Marshal the ntfy message to JSON
	payload, err := json.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= w.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: wait 2^attempt seconds, max 30 seconds
			backoffSeconds := math.Min(math.Pow(2, float64(attempt)), 30)
			slog.Debug("retrying webhook request", "attempt", attempt, "backoff_seconds", backoffSeconds)
			time.Sleep(time.Duration(backoffSeconds) * time.Second)
		}

		// Create HTTP request for each attempt to avoid body consumption issues
		req, err := http.NewRequest(http.MethodPost, w.webhookURL, bytes.NewBuffer(payload))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "ntfy-to-slack/"+version.Get().Version)

		// Make the request
		resp, err := w.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("webhook request failed: %w", err)
			slog.Debug("webhook request failed", "attempt", attempt+1, "err", err)
			continue
		}

		// Check status code
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024)) // Limit error response to 1KB
			resp.Body.Close()
			lastErr = fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, string(body))

			// Don't retry on client errors (4xx), only on server errors (5xx) and network issues
			if resp.StatusCode >= 400 && resp.StatusCode < 500 {
				slog.Debug("webhook client error, not retrying", "status", resp.StatusCode)
				return nil, fmt.Errorf("webhook post-processing failed (%s): %w", w.webhookURL, lastErr)
			}

			slog.Debug("webhook server error, will retry", "status", resp.StatusCode, "attempt", attempt+1)
			continue
		}

		// Limit response size to prevent memory exhaustion
		maxBytes := int64(w.maxResponseSizeMB * 1024 * 1024)
		limitedReader := io.LimitReader(resp.Body, maxBytes)

		// Read response
		body, err := io.ReadAll(limitedReader)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("failed to read response: %w", err)
			slog.Debug("failed to read webhook response", "attempt", attempt+1, "err", err)
			continue
		}

		// Check if response was truncated
		if int64(len(body)) >= maxBytes {
			slog.Warn("webhook response truncated due to size limit", "max_size_mb", w.maxResponseSizeMB)
		}

		// Try to unmarshal as slack message
		var slackMsg SlackMessage
		if err := json.Unmarshal(body, &slackMsg); err != nil {
			// If it fails, treat the response as plain text
			slackMsg.Text = string(body)
			slog.Debug("webhook response treated as plain text", "response", string(body))
		}

		// Slack rejects a message with no text (the "no_text" error). This is
		// a malformed response, not a transient failure -- retrying the same
		// webhook with the same input won't produce different content -- so
		// report it the same way a parse failure is reported above, rather
		// than retrying or silently sending something Slack will reject.
		if strings.TrimSpace(slackMsg.Text) == "" {
			return nil, fmt.Errorf("webhook post-processing failed (%s): response produced an empty Slack message text", w.webhookURL)
		}

		slog.Debug("webhook request successful", "attempt", attempt+1, "response_size", len(body))
		return &slackMsg, nil
	}

	// maxRetries+1 is the number of attempts made, not the number of retries.
	return nil, fmt.Errorf("webhook post-processing failed after %d attempts (%s): %w", w.maxRetries+1, w.webhookURL, lastErr)
}
