// Package testutil holds test doubles shared across this module's test
// files. It's a regular (non-_test.go) package so that test files in
// different directories -- internal/app, internal/config, internal/ntfy,
// internal/processor, internal/slack -- can all import the same mocks
// instead of each defining their own copy, which is what happened when the
// test suite lived entirely under tests/unit and tests/integration as two
// flat, unrelated package trees.
package testutil

import (
	"errors"
	"net/http"

	"github.com/ozskywalker/ntfy-to-slack/internal/config"
)

// MockHTTPClient implements config.HTTPClient (and, transitively,
// processor.HTTPClient and ntfy's dependency on the same shape) for tests
// that need to control or observe outgoing HTTP requests without a real
// network call.
type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if m.DoFunc != nil {
		return m.DoFunc(req)
	}
	return nil, errors.New("DoFunc not implemented")
}

// MockConfigProvider implements config.Provider for tests that need to
// construct an App or exercise config-consuming code without going through
// config.New's flag/env parsing.
type MockConfigProvider struct {
	Domain                   string
	Topic                    string
	Auth                     string
	Username                 string
	Password                 string
	WebhookURL               string
	PostProcessWebhook       string
	PostProcessTemplate      string
	PostProcessTemplateFile  string
	WebhookTimeoutSeconds    int
	WebhookRetries           int
	WebhookMaxResponseSizeMB int
}

func (m *MockConfigProvider) GetNtfyDomain() string              { return m.Domain }
func (m *MockConfigProvider) GetNtfyTopic() string               { return m.Topic }
func (m *MockConfigProvider) GetNtfyAuth() string                { return m.Auth }
func (m *MockConfigProvider) GetNtfyUsername() string            { return m.Username }
func (m *MockConfigProvider) GetNtfyPassword() string            { return m.Password }
func (m *MockConfigProvider) GetSlackWebhookURL() string         { return m.WebhookURL }
func (m *MockConfigProvider) GetPostProcessWebhook() string      { return m.PostProcessWebhook }
func (m *MockConfigProvider) GetPostProcessTemplate() string     { return m.PostProcessTemplate }
func (m *MockConfigProvider) GetPostProcessTemplateFile() string { return m.PostProcessTemplateFile }
func (m *MockConfigProvider) GetWebhookTimeoutSeconds() int      { return m.WebhookTimeoutSeconds }
func (m *MockConfigProvider) GetWebhookRetries() int             { return m.WebhookRetries }
func (m *MockConfigProvider) GetWebhookMaxResponseSizeMB() int   { return m.WebhookMaxResponseSizeMB }
func (m *MockConfigProvider) Validate() error                    { return nil }

// MockMessageSender implements processor.MessageSender for tests
// exercising MessageProcessor without a real Slack call.
type MockMessageSender struct {
	SentMessages []config.SlackMessage
	SendError    error
}

func (m *MockMessageSender) Send(message *config.SlackMessage) error {
	if m.SendError != nil {
		return m.SendError
	}
	if message != nil {
		m.SentMessages = append(m.SentMessages, *message)
	}
	return nil
}

// MockPostProcessor implements config.PostProcessor for tests exercising
// MessageProcessor's post-processing path without a real template or
// webhook call.
type MockPostProcessor struct {
	ProcessFunc func(message *config.NtfyMessage) (*config.SlackMessage, error)
	CallCount   int
}

func (m *MockPostProcessor) Process(message *config.NtfyMessage) (*config.SlackMessage, error) {
	m.CallCount++
	if m.ProcessFunc != nil {
		return m.ProcessFunc(message)
	}
	return &config.SlackMessage{Text: "Processed: " + message.Message}, nil
}

// CustomErrorSender is a processor.MessageSender that returns a
// caller-scripted sequence of errors (nil for success) on successive Send
// calls, for tests asserting that message processing continues past
// individual delivery failures.
type CustomErrorSender struct {
	SentMessages []config.SlackMessage
	Errors       []error
	SendCount    *int
}

func (c *CustomErrorSender) Send(message *config.SlackMessage) error {
	defer func() { *c.SendCount++ }()
	if *c.SendCount < len(c.Errors) && c.Errors[*c.SendCount] != nil {
		return c.Errors[*c.SendCount]
	}
	if message != nil {
		c.SentMessages = append(c.SentMessages, *message)
	}
	return nil
}
