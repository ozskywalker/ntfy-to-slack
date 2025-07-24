package integration

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"path/filepath"

	"github.com/ozskywalker/ntfy-to-slack/internal/config"
)

// TestConfig_PostProcessorValidation tests post-processor configuration validation
// This is a proper integration test that creates real configurations and validates them
func TestConfig_PostProcessorValidation(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		env             map[string]string
		shouldError     bool
		errorContains   string
		description     string
	}{
		{
			name: "no post-processor configured",
			args: []string{
				"--ntfy-topic", "test-topic",
				"--slack-webhook", "https://hooks.slack.com/services/test",
			},
			shouldError: false,
			description: "Configuration with no post-processor should be valid",
		},
		{
			name: "only webhook configured via flags",
			args: []string{
				"--ntfy-topic", "test-topic",
				"--slack-webhook", "https://hooks.slack.com/services/test",
				"--post-process-webhook", "https://example.com/webhook",
			},
			shouldError: false,
			description: "Configuration with only webhook post-processor should be valid",
		},
		{
			name: "only webhook configured via environment",
			args: []string{
				"--ntfy-topic", "test-topic",
				"--slack-webhook", "https://hooks.slack.com/services/test",
			},
			env: map[string]string{
				"POST_PROCESS_WEBHOOK": "https://example.com/webhook",
			},
			shouldError: false,
			description: "Configuration with webhook via environment should be valid",
		},
		{
			name: "only inline template configured via flags",
			args: []string{
				"--ntfy-topic", "test-topic",
				"--slack-webhook", "https://hooks.slack.com/services/test",
				"--post-process-template", "{{.Title}}: {{.Message}}",
			},
			shouldError: false,
			description: "Configuration with only inline template should be valid",
		},
		{
			name: "only inline template configured via environment",
			args: []string{
				"--ntfy-topic", "test-topic",
				"--slack-webhook", "https://hooks.slack.com/services/test",
			},
			env: map[string]string{
				"POST_PROCESS_TEMPLATE": "{{.Title}}: {{.Message}}",
			},
			shouldError: false,
			description: "Configuration with template via environment should be valid",
		},
		{
			name: "webhook and inline template configured via flags",
			args: []string{
				"--ntfy-topic", "test-topic",
				"--slack-webhook", "https://hooks.slack.com/services/test",
				"--post-process-webhook", "https://example.com/webhook",
				"--post-process-template", "{{.Title}}: {{.Message}}",
			},
			shouldError:   true,
			errorContains: "only one post-processing option can be specified",
			description:   "Configuration with multiple post-processors should fail validation",
		},
		{
			name: "webhook and inline template configured via environment",
			args: []string{
				"--ntfy-topic", "test-topic",
				"--slack-webhook", "https://hooks.slack.com/services/test",
			},
			env: map[string]string{
				"POST_PROCESS_WEBHOOK":  "https://example.com/webhook",
				"POST_PROCESS_TEMPLATE": "{{.Title}}: {{.Message}}",
			},
			shouldError:   true,
			errorContains: "only one post-processing option can be specified",
			description:   "Configuration with multiple post-processors via env should fail validation",
		},
		{
			name: "invalid webhook URL format",
			args: []string{
				"--ntfy-topic", "test-topic",
				"--slack-webhook", "https://hooks.slack.com/services/test",
				"--post-process-webhook", "not-a-url",
			},
			shouldError:   true,
			errorContains: "invalid post-process webhook URL format",
			description:   "Configuration with invalid webhook URL should fail validation",
		},
		{
			name: "webhook with unsupported scheme",
			args: []string{
				"--ntfy-topic", "test-topic",
				"--slack-webhook", "https://hooks.slack.com/services/test",
				"--post-process-webhook", "ftp://example.com/webhook",
			},
			shouldError:   true,
			errorContains: "invalid post-process webhook URL format",
			description:   "Configuration with unsupported webhook scheme should fail validation",
		},
		{
			name: "webhook with http scheme (allowed)",
			args: []string{
				"--ntfy-topic", "test-topic",
				"--slack-webhook", "https://hooks.slack.com/services/test",
				"--post-process-webhook", "http://localhost:8080/webhook",
			},
			shouldError: false,
			description: "Configuration with HTTP webhook (localhost) should be valid",
		},
		{
			name: "webhook timeout configuration validation",
			args: []string{
				"--ntfy-topic", "test-topic",
				"--slack-webhook", "https://hooks.slack.com/services/test",
				"--post-process-webhook", "https://example.com/webhook",
				"--webhook-timeout", "500", // Above maximum
			},
			shouldError:   true,
			errorContains: "webhook timeout must be between 1 and 300 seconds",
			description:   "Configuration with invalid webhook timeout should fail validation",
		},
		{
			name: "webhook retries configuration validation",
			args: []string{
				"--ntfy-topic", "test-topic",
				"--slack-webhook", "https://hooks.slack.com/services/test",
				"--post-process-webhook", "https://example.com/webhook",
				"--webhook-retries", "15", // Above maximum
			},
			shouldError:   true,
			errorContains: "webhook retries must be between 0 and 10",
			description:   "Configuration with invalid webhook retries should fail validation",
		},
		{
			name: "webhook max response size configuration validation",
			args: []string{
				"--ntfy-topic", "test-topic",
				"--slack-webhook", "https://hooks.slack.com/services/test",
				"--post-process-webhook", "https://example.com/webhook",
				"--webhook-max-response-size", "150", // Above maximum
			},
			shouldError:   true,
			errorContains: "webhook max response size must be between 1 and 100 MB",
			description:   "Configuration with invalid webhook response size should fail validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment variables
			oldEnv := make(map[string]string)
			for key, value := range tt.env {
				oldEnv[key] = os.Getenv(key)
				os.Setenv(key, value)
			}
			defer func() {
				for key := range tt.env {
					if oldValue, exists := oldEnv[key]; exists {
						os.Setenv(key, oldValue)
					} else {
						os.Unsetenv(key)
					}
				}
			}()

			// Create configuration using the public API
			cfg, err := config.New(tt.args)
			if err != nil {
				t.Errorf("Failed to create config: %v", err)
				return
			}

			// Validate the configuration
			err = cfg.Validate()

			if tt.shouldError {
				if err == nil {
					t.Errorf("%s: Expected error but got none", tt.description)
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("%s: Expected error to contain %q, got: %v", tt.description, tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("%s: Unexpected error: %v", tt.description, err)
				}
			}
		})
	}
}

// TestConfig_PostProcessorGetters tests the getter methods for post-processor configuration
// This verifies that the public API correctly exposes configuration values
func TestConfig_PostProcessorGetters(t *testing.T) {
	tests := []struct {
		name                    string
		args                    []string
		env                     map[string]string
		expectedWebhook         string
		expectedTemplate        string
		expectedTemplateFile    string
		description             string
	}{
		{
			name: "webhook configuration via flags",
			args: []string{
				"--ntfy-topic", "test-topic",
				"--slack-webhook", "https://hooks.slack.com/services/test",
				"--post-process-webhook", "https://example.com/webhook",
			},
			expectedWebhook: "https://example.com/webhook",
			description:     "Should correctly expose webhook URL configured via flags",
		},
		{
			name: "template configuration via flags",
			args: []string{
				"--ntfy-topic", "test-topic",
				"--slack-webhook", "https://hooks.slack.com/services/test",
				"--post-process-template", "{{.Title}}: {{.Message}}",
			},
			expectedTemplate: "{{.Title}}: {{.Message}}",
			description:      "Should correctly expose template configured via flags",
		},
		{
			name: "template file configuration via flags",
			args: []string{
				"--ntfy-topic", "test-topic",
				"--slack-webhook", "https://hooks.slack.com/services/test",
				"--post-process-template-file", "/path/to/template.tmpl",
			},
			expectedTemplateFile: "/path/to/template.tmpl",
			description:          "Should correctly expose template file path configured via flags",
		},
		{
			name: "webhook configuration via environment",
			args: []string{
				"--ntfy-topic", "test-topic",
				"--slack-webhook", "https://hooks.slack.com/services/test",
			},
			env: map[string]string{
				"POST_PROCESS_WEBHOOK": "https://env.example.com/webhook",
			},
			expectedWebhook: "https://env.example.com/webhook",
			description:     "Should correctly expose webhook URL configured via environment",
		},
		{
			name: "template configuration via environment",
			args: []string{
				"--ntfy-topic", "test-topic",
				"--slack-webhook", "https://hooks.slack.com/services/test",
			},
			env: map[string]string{
				"POST_PROCESS_TEMPLATE": "{{.Title}} - {{.Message}}",
			},
			expectedTemplate: "{{.Title}} - {{.Message}}",
			description:      "Should correctly expose template configured via environment",
		},
		{
			name: "template file configuration via environment",
			args: []string{
				"--ntfy-topic", "test-topic",
				"--slack-webhook", "https://hooks.slack.com/services/test",
			},
			env: map[string]string{
				"POST_PROCESS_TEMPLATE_FILE": "/env/path/to/template.tmpl",
			},
			expectedTemplateFile: "/env/path/to/template.tmpl",
			description:          "Should correctly expose template file path configured via environment",
		},
		{
			name: "flags override environment variables",
			args: []string{
				"--ntfy-topic", "test-topic",
				"--slack-webhook", "https://hooks.slack.com/services/test",
				"--post-process-webhook", "https://flag.example.com/webhook",
			},
			env: map[string]string{
				"POST_PROCESS_WEBHOOK": "https://env.example.com/webhook",
			},
			expectedWebhook: "https://flag.example.com/webhook",
			description:     "Should prefer flags over environment variables",
		},
		{
			name: "empty configuration",
			args: []string{
				"--ntfy-topic", "test-topic",
				"--slack-webhook", "https://hooks.slack.com/services/test",
			},
			expectedWebhook:      "",
			expectedTemplate:     "",
			expectedTemplateFile: "",
			description:          "Should return empty strings when no post-processor is configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment variables
			oldEnv := make(map[string]string)
			for key, value := range tt.env {
				oldEnv[key] = os.Getenv(key)
				os.Setenv(key, value)
			}
			defer func() {
				for key := range tt.env {
					if oldValue, exists := oldEnv[key]; exists {
						os.Setenv(key, oldValue)
					} else {
						os.Unsetenv(key)
					}
				}
			}()

			// Create configuration using the public API
			cfg, err := config.New(tt.args)
			if err != nil {
				t.Errorf("Failed to create config: %v", err)
				return
			}

			// Test getter methods using the Provider interface
			var provider config.Provider = cfg

			if got := provider.GetPostProcessWebhook(); got != tt.expectedWebhook {
				t.Errorf("%s: GetPostProcessWebhook() = %q, want %q", tt.description, got, tt.expectedWebhook)
			}

			if got := provider.GetPostProcessTemplate(); got != tt.expectedTemplate {
				t.Errorf("%s: GetPostProcessTemplate() = %q, want %q", tt.description, got, tt.expectedTemplate)
			}

			if got := provider.GetPostProcessTemplateFile(); got != tt.expectedTemplateFile {
				t.Errorf("%s: GetPostProcessTemplateFile() = %q, want %q", tt.description, got, tt.expectedTemplateFile)
			}
		})
	}
}

// TestConfig_TemplateFileIntegration tests template file loading and validation
// This tests real file system operations for template file configuration
func TestConfig_TemplateFileIntegration(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := ioutil.TempDir("", "ntfy-to-slack-config-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create valid template file
	validTemplatePath := filepath.Join(tempDir, "valid-template.tmpl")
	validTemplateContent := "🚨 Alert: {{.Title}}\n📄 Details: {{.Message}}\n🕐 Time: {{.Time}}"
	err = ioutil.WriteFile(validTemplatePath, []byte(validTemplateContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create valid template file: %v", err)
	}

	// Create invalid template file (for future validation)
	invalidTemplatePath := filepath.Join(tempDir, "invalid-template.tmpl")
	invalidTemplateContent := "{{.Title"
	err = ioutil.WriteFile(invalidTemplatePath, []byte(invalidTemplateContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid template file: %v", err)
	}

	// Non-existent file path (for future use in tests)
	_ = filepath.Join(tempDir, "non-existent.tmpl")

	tests := []struct {
		name              string
		templateFilePath  string
		shouldError       bool
		errorContains     string
		expectedPath      string
		description       string
	}{
		{
			name:             "valid template file via flags",
			templateFilePath: validTemplatePath,
			shouldError:      false,
			expectedPath:     validTemplatePath,
			description:      "Should accept valid template file path",
		},
		{
			name:             "template file configured via environment",
			templateFilePath: validTemplatePath,
			shouldError:      false,
			expectedPath:     validTemplatePath,
			description:      "Should accept template file path from environment",
		},
		{
			name:             "relative template file path",
			templateFilePath: "./tests/fixtures/template.tmpl", // This won't exist but path validation should work
			shouldError:      false,
			expectedPath:     "./tests/fixtures/template.tmpl",
			description:      "Should accept relative template file paths",
		},
		{
			name:             "absolute template file path",
			templateFilePath: validTemplatePath,
			shouldError:      false,
			expectedPath:     validTemplatePath,
			description:      "Should accept absolute template file paths",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test via command line flags
			args := []string{
				"--ntfy-topic", "test-topic",
				"--slack-webhook", "https://hooks.slack.com/services/test",
				"--post-process-template-file", tt.templateFilePath,
			}

			cfg, err := config.New(args)
			if err != nil {
				t.Errorf("Failed to create config: %v", err)
				return
			}

			// Validation should succeed for configuration creation (file existence check happens later)
			err = cfg.Validate()
			if tt.shouldError {
				if err == nil {
					t.Errorf("%s: Expected error but got none", tt.description)
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("%s: Expected error to contain %q, got: %v", tt.description, tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("%s: Unexpected error: %v", tt.description, err)
				}

				// Test that the file path is correctly stored
				if got := cfg.GetPostProcessTemplateFile(); got != tt.expectedPath {
					t.Errorf("%s: GetPostProcessTemplateFile() = %q, want %q", tt.description, got, tt.expectedPath)
				}
			}
		})

		// Test the same configuration via environment variables
		t.Run(tt.name+"_via_env", func(t *testing.T) {
			// Set up environment
			oldEnv := os.Getenv("POST_PROCESS_TEMPLATE_FILE")
			os.Setenv("POST_PROCESS_TEMPLATE_FILE", tt.templateFilePath)
			defer func() {
				if oldEnv != "" {
					os.Setenv("POST_PROCESS_TEMPLATE_FILE", oldEnv)
				} else {
					os.Unsetenv("POST_PROCESS_TEMPLATE_FILE")
				}
			}()

			args := []string{
				"--ntfy-topic", "test-topic",
				"--slack-webhook", "https://hooks.slack.com/services/test",
			}

			cfg, err := config.New(args)
			if err != nil {
				t.Errorf("Failed to create config: %v", err)
				return
			}

			err = cfg.Validate()
			if tt.shouldError {
				if err == nil {
					t.Errorf("%s (env): Expected error but got none", tt.description)
				}
			} else {
				if err != nil {
					t.Errorf("%s (env): Unexpected error: %v", tt.description, err)
				}

				// Test that the file path is correctly loaded from environment
				if got := cfg.GetPostProcessTemplateFile(); got != tt.expectedPath {
					t.Errorf("%s (env): GetPostProcessTemplateFile() = %q, want %q", tt.description, got, tt.expectedPath)
				}
			}
		})
	}
}

// TestConfig_WebhookConfigurationIntegration tests complete webhook configuration scenarios
// This tests realistic webhook configuration setups with all related parameters
func TestConfig_WebhookConfigurationIntegration(t *testing.T) {
	tests := []struct {
		name                     string
		args                     []string
		env                      map[string]string
		expectedWebhook          string
		expectedTimeout          int
		expectedRetries          int
		expectedMaxResponseSize  int
		shouldError              bool
		errorContains            string
		description              string
	}{
		{
			name: "webhook with default configuration via flags",
			args: []string{
				"--ntfy-topic", "test-topic",
				"--slack-webhook", "https://hooks.slack.com/services/test",
				"--post-process-webhook", "https://api.example.com/process",
			},
			expectedWebhook:         "https://api.example.com/process",
			expectedTimeout:         30,  // default
			expectedRetries:         3,   // default
			expectedMaxResponseSize: 1,   // default
			shouldError:             false,
			description:             "Should use default values for webhook parameters",
		},
		{
			name: "webhook with custom configuration via flags",
			args: []string{
				"--ntfy-topic", "test-topic",
				"--slack-webhook", "https://hooks.slack.com/services/test",
				"--post-process-webhook", "https://api.example.com/process",
				"--webhook-timeout", "60",
				"--webhook-retries", "5",
				"--webhook-max-response-size", "10",
			},
			expectedWebhook:         "https://api.example.com/process",
			expectedTimeout:         60,
			expectedRetries:         5,
			expectedMaxResponseSize: 10,
			shouldError:             false,
			description:             "Should use custom values for webhook parameters",
		},
		{
			name: "webhook configuration via environment",
			args: []string{
				"--ntfy-topic", "test-topic",
				"--slack-webhook", "https://hooks.slack.com/services/test",
			},
			env: map[string]string{
				"POST_PROCESS_WEBHOOK":             "https://env.example.com/process",
				"WEBHOOK_TIMEOUT_SECONDS":          "45",
				"WEBHOOK_RETRIES":                  "2",
				"WEBHOOK_MAX_RESPONSE_SIZE_MB":     "5",
			},
			expectedWebhook:         "https://env.example.com/process",
			expectedTimeout:         45,
			expectedRetries:         2,
			expectedMaxResponseSize: 5,
			shouldError:             false,
			description:             "Should correctly load webhook configuration from environment",
		},
		{
			name: "flags override environment for webhook config",
			args: []string{
				"--ntfy-topic", "test-topic",
				"--slack-webhook", "https://hooks.slack.com/services/test",
				"--post-process-webhook", "https://flag.example.com/process",
				"--webhook-timeout", "90",
			},
			env: map[string]string{
				"POST_PROCESS_WEBHOOK":         "https://env.example.com/process",
				"WEBHOOK_TIMEOUT_SECONDS":      "120",
				"WEBHOOK_RETRIES":              "2",
				"WEBHOOK_MAX_RESPONSE_SIZE_MB": "3",
			},
			expectedWebhook:         "https://flag.example.com/process",
			expectedTimeout:         90, // flag overrides env
			expectedRetries:         2,  // from env (not overridden by flag)
			expectedMaxResponseSize: 3,  // from env (not overridden by flag)
			shouldError:             false,
			description:             "Should prefer flag values over environment variables",
		},
		{
			name: "webhook timeout boundary validation - minimum",
			args: []string{
				"--ntfy-topic", "test-topic",
				"--slack-webhook", "https://hooks.slack.com/services/test",
				"--post-process-webhook", "https://api.example.com/process",
				"--webhook-timeout", "1",
			},
			expectedWebhook:         "https://api.example.com/process",
			expectedTimeout:         1,
			expectedRetries:         3, // default value
			expectedMaxResponseSize: 1, // default value
			shouldError:             false,
			description:             "Should accept minimum valid timeout value",
		},
		{
			name: "webhook timeout boundary validation - maximum",
			args: []string{
				"--ntfy-topic", "test-topic",
				"--slack-webhook", "https://hooks.slack.com/services/test",
				"--post-process-webhook", "https://api.example.com/process",
				"--webhook-timeout", "300",
			},
			expectedWebhook:         "https://api.example.com/process",
			expectedTimeout:         300,
			expectedRetries:         3, // default value
			expectedMaxResponseSize: 1, // default value
			shouldError:             false,
			description:             "Should accept maximum valid timeout value",
		},
		{
			name: "webhook retries boundary validation - minimum",
			args: []string{
				"--ntfy-topic", "test-topic",
				"--slack-webhook", "https://hooks.slack.com/services/test",
				"--post-process-webhook", "https://api.example.com/process",
				"--webhook-retries", "0",
			},
			expectedWebhook:         "https://api.example.com/process",
			expectedTimeout:         30, // default value
			expectedRetries:         0,
			expectedMaxResponseSize: 1, // default value
			shouldError:             false,
			description:             "Should accept minimum valid retries value (0)",
		},
		{
			name: "webhook retries boundary validation - maximum",
			args: []string{
				"--ntfy-topic", "test-topic",
				"--slack-webhook", "https://hooks.slack.com/services/test",
				"--post-process-webhook", "https://api.example.com/process",
				"--webhook-retries", "10",
			},
			expectedWebhook:         "https://api.example.com/process",
			expectedTimeout:         30, // default value
			expectedRetries:         10,
			expectedMaxResponseSize: 1, // default value
			shouldError:             false,
			description:             "Should accept maximum valid retries value",
		},
		{
			name: "webhook response size boundary validation - minimum",
			args: []string{
				"--ntfy-topic", "test-topic",
				"--slack-webhook", "https://hooks.slack.com/services/test",
				"--post-process-webhook", "https://api.example.com/process",
				"--webhook-max-response-size", "1",
			},
			expectedWebhook:         "https://api.example.com/process",
			expectedTimeout:         30, // default value
			expectedRetries:         3,  // default value
			expectedMaxResponseSize: 1,
			shouldError:             false,
			description:             "Should accept minimum valid response size value",
		},
		{
			name: "webhook response size boundary validation - maximum",
			args: []string{
				"--ntfy-topic", "test-topic",
				"--slack-webhook", "https://hooks.slack.com/services/test",
				"--post-process-webhook", "https://api.example.com/process",
				"--webhook-max-response-size", "100",
			},
			expectedWebhook:         "https://api.example.com/process",
			expectedTimeout:         30, // default value
			expectedRetries:         3,  // default value
			expectedMaxResponseSize: 100,
			shouldError:             false,
			description:             "Should accept maximum valid response size value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment variables
			oldEnv := make(map[string]string)
			for key, value := range tt.env {
				oldEnv[key] = os.Getenv(key)
				os.Setenv(key, value)
			}
			defer func() {
				for key := range tt.env {
					if oldValue, exists := oldEnv[key]; exists {
						os.Setenv(key, oldValue)
					} else {
						os.Unsetenv(key)
					}
				}
			}()

			// Create configuration
			cfg, err := config.New(tt.args)
			if err != nil {
				t.Errorf("Failed to create config: %v", err)
				return
			}

			// Validate configuration
			err = cfg.Validate()
			if tt.shouldError {
				if err == nil {
					t.Errorf("%s: Expected error but got none", tt.description)
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("%s: Expected error to contain %q, got: %v", tt.description, tt.errorContains, err)
				}
				return
			}

			if err != nil {
				t.Errorf("%s: Unexpected error: %v", tt.description, err)
				return
			}

			// Test getter methods
			var provider config.Provider = cfg

			if got := provider.GetPostProcessWebhook(); got != tt.expectedWebhook {
				t.Errorf("%s: GetPostProcessWebhook() = %q, want %q", tt.description, got, tt.expectedWebhook)
			}

			if got := provider.GetWebhookTimeoutSeconds(); got != tt.expectedTimeout {
				t.Errorf("%s: GetWebhookTimeoutSeconds() = %d, want %d", tt.description, got, tt.expectedTimeout)
			}

			if got := provider.GetWebhookRetries(); got != tt.expectedRetries {
				t.Errorf("%s: GetWebhookRetries() = %d, want %d", tt.description, got, tt.expectedRetries)
			}

			if got := provider.GetWebhookMaxResponseSizeMB(); got != tt.expectedMaxResponseSize {
				t.Errorf("%s: GetWebhookMaxResponseSizeMB() = %d, want %d", tt.description, got, tt.expectedMaxResponseSize)
			}
		})
	}
}

// TestConfig_CompleteConfigurationScenarios tests realistic complete configuration scenarios
// This tests the end-to-end configuration workflow including all aspects
func TestConfig_CompleteConfigurationScenarios(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		env         map[string]string
		shouldError bool
		description string
		scenario    string
	}{
		{
			name: "production-like configuration via flags",
			args: []string{
				"--ntfy-domain", "notifications.company.com",
				"--ntfy-topic", "critical-alerts",
				"--ntfy-auth", "tk_production_token_123456789",
				"--slack-webhook", "https://hooks.slack.com/services/T1234567890/B1234567890/XXXXXXXXXXXXXXXXXXXXXXXX",
				"--post-process-webhook", "https://api.company.com/process-alert",
				"--webhook-timeout", "60",
				"--webhook-retries", "5",
				"--webhook-max-response-size", "10",
			},
			shouldError: false,
			description: "Should handle production-like configuration with all options",
			scenario:    "production",
		},
		{
			name: "development configuration via environment",
			args: []string{},
			env: map[string]string{
				"NTFY_DOMAIN":                      "dev.ntfy.sh",
				"NTFY_TOPIC":                       "dev-testing",
				"SLACK_WEBHOOK_URL":                "https://hooks.slack.com/services/DEV/DEV/DEVTOKENXXXXXXXXXXXXXXX",
				"POST_PROCESS_TEMPLATE":            "🔧 DEV: {{.Title}} - {{.Message}}",
				"WEBHOOK_TIMEOUT_SECONDS":          "10",
				"WEBHOOK_RETRIES":                  "1",
				"WEBHOOK_MAX_RESPONSE_SIZE_MB":     "1",
			},
			shouldError: false,
			description: "Should handle development configuration via environment",
			scenario:    "development",
		},
		{
			name: "minimal valid configuration",
			args: []string{
				"--ntfy-topic", "simple-alerts",
				"--slack-webhook", "https://hooks.slack.com/services/MINIMAL/MINIMAL/MINIMALTOKEN",
			},
			shouldError: false,
			description: "Should handle minimal valid configuration with defaults",
			scenario:    "minimal",
		},
		{
			name: "mixed configuration sources",
			args: []string{
				"--ntfy-topic", "mixed-alerts", // flag
				"--slack-webhook", "https://hooks.slack.com/services/MIXED/MIXED/MIXEDTOKEN", // flag
				"--webhook-timeout", "90", // flag overrides env
			},
			env: map[string]string{
				"NTFY_DOMAIN":                  "env.ntfy.sh", // env
				"POST_PROCESS_WEBHOOK":         "https://env.api.com/process", // env
				"WEBHOOK_TIMEOUT_SECONDS":      "120", // env (overridden by flag)
				"WEBHOOK_RETRIES":              "7", // env
				"WEBHOOK_MAX_RESPONSE_SIZE_MB": "25", // env
			},
			shouldError: false,
			description: "Should handle mixed configuration sources with proper precedence",
			scenario:    "mixed",
		},
		{
			name: "configuration with edge case values",
			args: []string{
				"--ntfy-domain", "example.co", // minimum valid domain
				"--ntfy-topic", "a", // minimum valid topic
				"--slack-webhook", "https://hooks.slack.com/services/A/B/C",
				"--post-process-webhook", "http://127.0.0.1:8080/webhook", // localhost HTTP
				"--webhook-timeout", "1", // minimum
				"--webhook-retries", "0", // minimum
				"--webhook-max-response-size", "1", // minimum
			},
			shouldError: false,
			description: "Should handle edge case valid values",
			scenario:    "edge-cases",
		},
		{
			name: "maximum configuration values",
			args: []string{
				"--ntfy-domain", "very.long.subdomain.example.company.notifications.system.com",
				"--ntfy-topic", "maximum_length_topic_name_with_exactly_64_characters_1234567890", // 64 chars
				"--slack-webhook", "https://hooks.slack.com/services/TXXXXXXXXXXXXXXX/BXXXXXXXXXXXXXXX/XXXXXXXXXXXXXXXXXXXXXXXX",
				"--post-process-webhook", "https://very.long.api.endpoint.example.company.com/api/v1/notifications/process",
				"--webhook-timeout", "300", // maximum
				"--webhook-retries", "10", // maximum
				"--webhook-max-response-size", "100", // maximum
			},
			shouldError: false,
			description: "Should handle maximum valid configuration values",
			scenario:    "maximum",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment variables
			oldEnv := make(map[string]string)
			for key, value := range tt.env {
				oldEnv[key] = os.Getenv(key)
				os.Setenv(key, value)
			}
			defer func() {
				for key := range tt.env {
					if oldValue, exists := oldEnv[key]; exists {
						os.Setenv(key, oldValue)
					} else {
						os.Unsetenv(key)
					}
				}
			}()

			// Create and validate configuration
			cfg, err := config.New(tt.args)
			if err != nil {
				t.Errorf("Failed to create config for %s scenario: %v", tt.scenario, err)
				return
			}

			err = cfg.Validate()
			if tt.shouldError {
				if err == nil {
					t.Errorf("%s: Expected error but got none", tt.description)
				}
				return
			}

			if err != nil {
				t.Errorf("%s: Unexpected validation error: %v", tt.description, err)
				return
			}

			// Verify that the configuration implements the Provider interface correctly
			var provider config.Provider = cfg

			// Basic validation that all getters work
			domain := provider.GetNtfyDomain()
			topic := provider.GetNtfyTopic()
			slackURL := provider.GetSlackWebhookURL()

			if domain == "" {
				t.Errorf("%s: GetNtfyDomain() returned empty string", tt.description)
			}
			if topic == "" {
				t.Errorf("%s: GetNtfyTopic() returned empty string", tt.description)
			}
			if slackURL == "" {
				t.Errorf("%s: GetSlackWebhookURL() returned empty string", tt.description)
			}

			// Test that webhook configuration is consistent
			webhookURL := provider.GetPostProcessWebhook()
			if webhookURL != "" {
				// If webhook is configured, all webhook settings should be valid
				timeout := provider.GetWebhookTimeoutSeconds()
				retries := provider.GetWebhookRetries()
				maxSize := provider.GetWebhookMaxResponseSizeMB()

				if timeout < 1 || timeout > 300 {
					t.Errorf("%s: Invalid webhook timeout: %d", tt.description, timeout)
				}
				if retries < 0 || retries > 10 {
					t.Errorf("%s: Invalid webhook retries: %d", tt.description, retries)
				}
				if maxSize < 1 || maxSize > 100 {
					t.Errorf("%s: Invalid webhook max response size: %d", tt.description, maxSize)
				}
			}

			t.Logf("%s scenario validated successfully", tt.scenario)
		})
	}
}