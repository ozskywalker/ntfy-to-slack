package unit_test

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/ozskywalker/ntfy-to-slack/internal/config"
)

// MockHTTPClient implements config.HTTPClient interface for testing
type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if m.DoFunc != nil {
		return m.DoFunc(req)
	}
	return nil, errors.New("DoFunc not implemented")
}

// MockConfigProvider implements config.Provider interface for testing
type MockConfigProvider struct {
	Domain                   string
	Topic                    string
	Auth                     string
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
func (m *MockConfigProvider) GetSlackWebhookURL() string         { return m.WebhookURL }
func (m *MockConfigProvider) GetPostProcessWebhook() string      { return m.PostProcessWebhook }
func (m *MockConfigProvider) GetPostProcessTemplate() string     { return m.PostProcessTemplate }
func (m *MockConfigProvider) GetPostProcessTemplateFile() string { return m.PostProcessTemplateFile }
func (m *MockConfigProvider) GetWebhookTimeoutSeconds() int      { return m.WebhookTimeoutSeconds }
func (m *MockConfigProvider) GetWebhookRetries() int             { return m.WebhookRetries }
func (m *MockConfigProvider) GetWebhookMaxResponseSizeMB() int   { return m.WebhookMaxResponseSizeMB }
func (m *MockConfigProvider) Validate() error                   { return nil }

// MockNtfyClient implements ntfy.Client interface for testing
type MockNtfyClient struct {
	ConnectFunc func() (io.ReadCloser, error)
}

func (m *MockNtfyClient) Connect() (io.ReadCloser, error) {
	if m.ConnectFunc != nil {
		return m.ConnectFunc()
	}
	return io.NopCloser(strings.NewReader("")), nil
}

// MockStreamProcessor implements processor.StreamProcessor interface for testing
type MockStreamProcessor struct {
	ProcessStreamFunc func(reader io.Reader) error
	ProcessedInputs   []string
}

func (m *MockStreamProcessor) ProcessStream(reader io.Reader) error {
	if m.ProcessStreamFunc != nil {
		return m.ProcessStreamFunc(reader)
	}
	
	// Capture input for verification
	if data, err := io.ReadAll(reader); err == nil {
		m.ProcessedInputs = append(m.ProcessedInputs, string(data))
	}
	
	return nil
}

// MockMessageSender implements processor.MessageSender interface for testing
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

// MockPostProcessor for testing
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

// CustomErrorSender is a test helper that simulates send errors
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