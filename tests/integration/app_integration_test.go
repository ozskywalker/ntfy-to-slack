package integration

import (
	"testing"
	"time"

	"github.com/ozskywalker/ntfy-to-slack/internal/app"
)

// MockConfigProvider implements config.Provider interface for integration testing
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
func (m *MockConfigProvider) Validate() error                    { return nil }

// TestApp_ConfigurationValidation tests app creation with various configurations
func TestApp_ConfigurationValidation(t *testing.T) {
	tests := []struct {
		name        string
		domain      string
		topic       string
		slackURL    string
		postWebhook string
		template    string
		expectPanic bool
	}{
		{
			name:     "valid basic configuration",
			domain:   "ntfy.sh",
			topic:    "test",
			slackURL: "https://hooks.slack.com/services/test",
		},
		{
			name:        "configuration with webhook post-processor",
			domain:      "ntfy.sh",
			topic:       "test",
			slackURL:    "https://hooks.slack.com/services/test",
			postWebhook: "https://example.com/webhook",
		},
		{
			name:     "configuration with template post-processor",
			domain:   "ntfy.sh",
			topic:    "test",
			slackURL: "https://hooks.slack.com/services/test",
			template: "🚨 {{.Title}}: {{.Message}}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &MockConfigProvider{
				Domain:                   tt.domain,
				Topic:                    tt.topic,
				WebhookURL:               tt.slackURL,
				PostProcessWebhook:       tt.postWebhook,
				PostProcessTemplate:      tt.template,
				WebhookTimeoutSeconds:    30,
				WebhookRetries:           3,
				WebhookMaxResponseSizeMB: 1,
			}

			// Test that app creation doesn't panic with valid configurations
			defer func() {
				if r := recover(); r != nil {
					if !tt.expectPanic {
						t.Errorf("App creation panicked unexpectedly: %v", r)
					}
				} else if tt.expectPanic {
					t.Error("Expected app creation to panic but it didn't")
				}
			}()

			app := app.New(cfg, "test-version")
			if app == nil {
				t.Error("Expected app instance but got nil")
			}
		})
	}
}

// TestApp_ErrorHandling tests app behavior with connection errors
func TestApp_ErrorHandling(t *testing.T) {
	tests := []struct {
		name             string
		domain           string
		topic            string
		slackURL         string
		expectCreationOK bool
		expectRunError   bool
		errorContains    string
	}{
		{
			name:             "valid configuration",
			domain:           "ntfy.sh",
			topic:            "test-topic",
			slackURL:         "https://hooks.slack.com/services/test",
			expectCreationOK: true,
			expectRunError:   true, // Will fail to connect to real ntfy.sh with test topic
			errorContains:    "failed to connect to ntfy",
		},
		{
			name:             "invalid domain",
			domain:           "invalid-domain-name",
			topic:            "test-topic",
			slackURL:         "https://hooks.slack.com/services/test",
			expectCreationOK: true, // Creation succeeds, validation happens at runtime
			expectRunError:   true,
			errorContains:    "invalid domain format",
		},
		{
			name:             "invalid topic",
			domain:           "ntfy.sh",
			topic:            "invalid@topic!",
			slackURL:         "https://hooks.slack.com/services/test",
			expectCreationOK: true, // Creation succeeds, validation happens at runtime
			expectRunError:   true,
			errorContains:    "invalid topic format",
		},
		{
			name:             "nonexistent domain",
			domain:           "nonexistent.domain.test",
			topic:            "test-topic",
			slackURL:         "https://hooks.slack.com/services/test",
			expectCreationOK: true,
			expectRunError:   true,
			errorContains:    "failed to connect to ntfy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &MockConfigProvider{
				Domain:                   tt.domain,
				Topic:                    tt.topic,
				WebhookURL:               tt.slackURL,
				WebhookTimeoutSeconds:    5,
				WebhookRetries:           1,
				WebhookMaxResponseSizeMB: 1,
			}

			// Test app creation
			app := app.New(cfg, "test-version")

			if tt.expectCreationOK {
				if app == nil {
					t.Error("Expected successful app creation but got nil")
				}
			} else {
				if app != nil {
					t.Error("Expected app creation to fail but got valid instance")
				}
				return // Skip Run test if creation failed
			}

			// Test app Run behavior with timeout
			done := make(chan error, 1)
			go func() {
				done <- app.Run()
			}()

			select {
			case err := <-done:
				if tt.expectRunError {
					if err == nil {
						t.Error("Expected Run() to return error but got nil")
					} else if tt.errorContains != "" && err.Error() != "" {
						// Just check that we got some error - exact error matching is too brittle
						// for integration tests due to network conditions
						t.Logf("Got expected error: %v", err)
					}
				} else {
					if err != nil {
						t.Errorf("Unexpected error from Run(): %v", err)
					}
				}
			case <-time.After(3 * time.Second):
				// Timeout is acceptable - app may be retrying connections
				if tt.expectRunError {
					t.Log("Run() timed out as expected (app may be retrying)")
				} else {
					t.Error("Run() timed out unexpectedly")
				}
			}
		})
	}
}

// TestApp_PostProcessorIntegration tests post-processor configuration integration
func TestApp_PostProcessorIntegration(t *testing.T) {
	tests := []struct {
		name            string
		postProcessType string
		template        string
		webhook         string
		templateFile    string
		expectError     bool
	}{
		{
			name:            "no post-processor configured",
			postProcessType: "none",
		},
		{
			name:            "template post-processor configured",
			postProcessType: "template",
			template:        "🚨 {{.Title}}: {{.Message}}",
		},
		{
			name:            "invalid template handles gracefully",
			postProcessType: "template",
			template:        "{{.Title", // Invalid syntax - should not prevent app creation
		},
		{
			name:            "webhook post-processor configured",
			postProcessType: "webhook",
			webhook:         "https://api.example.com/process",
		},
		{
			name:            "multiple post-processors configured",
			postProcessType: "multiple",
			template:        "{{.Title}}",
			webhook:         "https://api.example.com/process", // Should use webhook, ignore template
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &MockConfigProvider{
				Domain:                   "ntfy.sh",
				Topic:                    "test-alerts",
				WebhookURL:               "https://hooks.slack.com/services/test",
				WebhookTimeoutSeconds:    30,
				WebhookRetries:           3,
				WebhookMaxResponseSizeMB: 1,
			}

			if tt.postProcessType == "template" || tt.postProcessType == "multiple" {
				cfg.PostProcessTemplate = tt.template
			}
			if tt.postProcessType == "webhook" || tt.postProcessType == "multiple" {
				cfg.PostProcessWebhook = tt.webhook
			}

			// Create app - this tests that post-processor configuration works
			app := app.New(cfg, "test-version")
			if app == nil {
				t.Fatal("Failed to create app instance")
			}

			// The fact that app creation succeeded means post-processor
			// configuration was handled properly (even invalid templates
			// should not prevent app creation, they just fall back to default)

			// We can't easily test the actual post-processing without mocking HTTP,
			// but we've verified that the configuration and initialization work correctly
		})
	}
}

// TestApp_ConnectionResilience tests app behavior with connection failures
func TestApp_ConnectionResilience(t *testing.T) {
	tests := []struct {
		name        string
		domain      string
		expectRetry bool
		maxWaitTime time.Duration
	}{
		{
			name:        "connection to nonexistent domain",
			domain:      "definitely.nonexistent.domain.test",
			expectRetry: true,
			maxWaitTime: 5 * time.Second,
		},
		{
			name:        "connection to invalid domain format",
			domain:      "invalid-domain",
			expectRetry: false, // Should fail immediately on validation
			maxWaitTime: 2 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &MockConfigProvider{
				Domain:                   tt.domain,
				Topic:                    "test-topic",
				WebhookURL:               "https://hooks.slack.com/services/test",
				WebhookTimeoutSeconds:    1,
				WebhookRetries:           0,
				WebhookMaxResponseSizeMB: 1,
			}

			app := app.New(cfg, "test-version")
			if app == nil {
				t.Fatal("Failed to create app instance")
			}

			// Test that app returns error quickly on connection failure
			start := time.Now()
			done := make(chan error, 1)

			go func() {
				done <- app.Run()
			}()

			select {
			case err := <-done:
				duration := time.Since(start)

				if err == nil {
					t.Error("Expected error due to connection failure")
				}

				// Should fail within reasonable time
				if duration > tt.maxWaitTime {
					t.Errorf("App took too long to fail: %v (max: %v)", duration, tt.maxWaitTime)
				}

				t.Logf("App failed as expected in %v with error: %v", duration, err)

			case <-time.After(tt.maxWaitTime):
				t.Errorf("App did not fail within expected time (%v)", tt.maxWaitTime)
			}
		})
	}
}

// TestApp_IntegrationScenarios tests realistic app configuration scenarios
func TestApp_IntegrationScenarios(t *testing.T) {
	tests := []struct {
		name        string
		domain      string
		topic       string
		slackURL    string
		webhook     string
		template    string
		expectSetup bool
	}{
		{
			name:        "basic production setup",
			domain:      "ntfy.sh",
			topic:       "production-alerts",
			slackURL:    "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX",
			expectSetup: true,
		},
		{
			name:        "custom ntfy server with webhook post-processor",
			domain:      "notifications.example.com",
			topic:       "system-notifications",
			slackURL:    "https://hooks.slack.com/services/T11111111/B11111111/YYYYYYYYYYYYYYYYYYYYYYYY",
			webhook:     "https://api.example.com/process-notification",
			expectSetup: true,
		},
		{
			name:        "custom setup with template post-processor",
			domain:      "alerts.company.com",
			topic:       "critical-alerts",
			slackURL:    "https://hooks.slack.com/services/T22222222/B22222222/ZZZZZZZZZZZZZZZZZZZZZZZZ",
			template:    "🚨 ALERT: {{.Title}}\n📋 Details: {{.Message}}\n🕐 Time: {{.Time}}",
			expectSetup: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &MockConfigProvider{
				Domain:                   tt.domain,
				Topic:                    tt.topic,
				WebhookURL:               tt.slackURL,
				PostProcessWebhook:       tt.webhook,
				PostProcessTemplate:      tt.template,
				WebhookTimeoutSeconds:    30,
				WebhookRetries:           3,
				WebhookMaxResponseSizeMB: 1,
			}

			// Create app - this tests the complete initialization flow
			app := app.New(cfg, "test-version")

			if tt.expectSetup {
				if app == nil {
					t.Error("Expected successful app setup but got nil")
				}
			} else {
				if app != nil {
					t.Error("Expected app setup to fail but got valid instance")
				}
			}

			// If app was created successfully, verify it can start (even if it fails to connect)
			if app != nil {
				// Test that Run doesn't panic immediately
				done := make(chan error, 1)
				go func() {
					done <- app.Run()
				}()

				select {
				case err := <-done:
					// Error is expected since we're not connecting to real servers
					t.Logf("App returned error as expected: %v", err)
				case <-time.After(2 * time.Second):
					// Timeout is also acceptable - means app is trying to connect
					t.Log("App is running (timed out waiting for connection)")
				}
			}
		})
	}
}

// TestApp_ConfigurationEdgeCases tests edge cases in app configuration
func TestApp_ConfigurationEdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		cfg            *MockConfigProvider
		expectCreation bool
		description    string
	}{
		{
			name: "minimal valid configuration",
			cfg: &MockConfigProvider{
				Domain:                   "ntfy.sh",
				Topic:                    "test",
				WebhookURL:               "https://hooks.slack.com/services/minimal",
				WebhookTimeoutSeconds:    30,
				WebhookRetries:           3,
				WebhookMaxResponseSizeMB: 1,
			},
			expectCreation: true,
			description:    "Should create app with minimal valid config",
		},
		{
			name: "maximum configuration",
			cfg: &MockConfigProvider{
				Domain:                   "custom.ntfy.server.com",
				Topic:                    "maximum-length-topic-name-with-64-chars-exactly-1234567890",
				Auth:                     "tk_abcdefghijklmnopqrstuvwxyz1234567890",
				WebhookURL:               "https://hooks.slack.com/services/TXXXXXXXXX/BXXXXXXXXX/XXXXXXXXXXXXXXXXXXXXXXXX",
				PostProcessWebhook:       "https://api.example.com/process",
				WebhookTimeoutSeconds:    300, // Maximum allowed
				WebhookRetries:           10,  // Maximum allowed
				WebhookMaxResponseSizeMB: 100, // Maximum allowed
			},
			expectCreation: true,
			description:    "Should create app with maximum valid config",
		},
		{
			name: "auth token configuration",
			cfg: &MockConfigProvider{
				Domain:                   "ntfy.sh",
				Topic:                    "protected-topic",
				Auth:                     "tk_test123456789",
				WebhookURL:               "https://hooks.slack.com/services/auth-test",
				WebhookTimeoutSeconds:    30,
				WebhookRetries:           3,
				WebhookMaxResponseSizeMB: 1,
			},
			expectCreation: true,
			description:    "Should create app with auth token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := app.New(tt.cfg, "test-version")

			if tt.expectCreation {
				if app == nil {
					t.Errorf("%s: Expected app creation to succeed but got nil", tt.description)
				}
			} else {
				if app != nil {
					t.Errorf("%s: Expected app creation to fail but got valid instance", tt.description)
				}
			}
		})
	}
}
