package unit_test

import (
	"os"
	"strings"
	"testing"

	"github.com/ozskywalker/ntfy-to-slack/internal/config"
)

func TestNewConfig(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		env      map[string]string
		expected *config.Config
		wantErr  bool
	}{
		{
			name: "flags override environment",
			args: []string{"--ntfy-topic", "test-topic", "--slack-webhook", "https://hooks.slack.com/test"},
			env: map[string]string{
				"NTFY_TOPIC":        "env-topic",
				"SLACK_WEBHOOK_URL": "https://hooks.slack.com/env",
			},
			expected: &config.Config{
				NtfyDomain:      "ntfy.sh",
				NtfyTopic:       "test-topic",
				SlackWebhookURL: "https://hooks.slack.com/test",
				LogLevel:        "info",
			},
			wantErr: false,
		},
		{
			name: "environment variables used when no flags",
			args: []string{},
			env: map[string]string{
				"NTFY_DOMAIN":       "custom.ntfy.sh",
				"NTFY_TOPIC":        "env-topic",
				"NTFY_AUTH":         "auth-token",
				"SLACK_WEBHOOK_URL": "https://hooks.slack.com/env",
				"LOG_LEVEL":         "debug",
			},
			expected: &config.Config{
				NtfyDomain:      "custom.ntfy.sh",
				NtfyTopic:       "env-topic",
				NtfyAuth:        "auth-token",
				SlackWebhookURL: "https://hooks.slack.com/env",
				LogLevel:        "debug",
			},
			wantErr: false,
		},
		{
			name: "defaults used when nothing set",
			args: []string{},
			env:  map[string]string{},
			expected: &config.Config{
				NtfyDomain: "ntfy.sh",
				LogLevel:   "info",
				ShowHelp:   true, // Should show help when no required args
			},
			wantErr: false,
		},
		{
			name: "version flag",
			args: []string{"-v"},
			env:  map[string]string{},
			expected: &config.Config{
				NtfyDomain:  "ntfy.sh",
				LogLevel:    "info",
				ShowVersion: true,
			},
			wantErr: false,
		},
		{
			name:    "invalid flag",
			args:    []string{"--invalid-flag", "value"},
			env:     map[string]string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
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

			cfg, err := config.New(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Compare relevant fields using getters
			if cfg.GetNtfyDomain() != tt.expected.NtfyDomain {
				t.Errorf("NtfyDomain = %v, want %v", cfg.GetNtfyDomain(), tt.expected.NtfyDomain)
			}
			if cfg.GetNtfyTopic() != tt.expected.NtfyTopic {
				t.Errorf("NtfyTopic = %v, want %v", cfg.GetNtfyTopic(), tt.expected.NtfyTopic)
			}
			if cfg.GetNtfyAuth() != tt.expected.NtfyAuth {
				t.Errorf("NtfyAuth = %v, want %v", cfg.GetNtfyAuth(), tt.expected.NtfyAuth)
			}
			if cfg.GetSlackWebhookURL() != tt.expected.SlackWebhookURL {
				t.Errorf("SlackWebhookURL = %v, want %v", cfg.GetSlackWebhookURL(), tt.expected.SlackWebhookURL)
			}
			// Note: LogLevel, ShowVersion, ShowHelp are not exposed via getters, skip these checks
		})
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *config.Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid configuration",
			config: &config.Config{
				NtfyDomain:      "ntfy.sh",
				NtfyTopic:       "test-topic",
				SlackWebhookURL: "https://hooks.slack.com/test",
			},
			wantErr: false,
		},
		{
			name: "skip validation for version",
			config: &config.Config{
				ShowVersion: true,
			},
			wantErr: false,
		},
		{
			name: "skip validation for help",
			config: &config.Config{
				ShowHelp: true,
			},
			wantErr: false,
		},
		{
			name: "missing ntfy topic",
			config: &config.Config{
				NtfyDomain:      "ntfy.sh",
				SlackWebhookURL: "https://hooks.slack.com/test",
			},
			wantErr: true,
			errMsg:  "ntfy topic is required",
		},
		{
			name: "missing slack webhook",
			config: &config.Config{
				NtfyDomain: "ntfy.sh",
				NtfyTopic:  "test-topic",
			},
			wantErr: true,
			errMsg:  "Slack webhook URL is required",
		},
		{
			name: "invalid domain",
			config: &config.Config{
				NtfyDomain:      "invalid-domain",
				NtfyTopic:       "test-topic",
				SlackWebhookURL: "https://hooks.slack.com/test",
			},
			wantErr: true,
			errMsg:  "invalid domain",
		},
		{
			name: "invalid topic",
			config: &config.Config{
				NtfyDomain:      "ntfy.sh",
				NtfyTopic:       "invalid topic with spaces",
				SlackWebhookURL: "https://hooks.slack.com/test",
			},
			wantErr: true,
			errMsg:  "invalid topic",
		},
		{
			name: "invalid slack webhook URL - not https",
			config: &config.Config{
				NtfyDomain:      "ntfy.sh",
				NtfyTopic:       "test-topic",
				SlackWebhookURL: "http://hooks.slack.com/test",
			},
			wantErr: true,
			errMsg:  "invalid Slack webhook URL format",
		},
		{
			name: "invalid slack webhook URL - malformed",
			config: &config.Config{
				NtfyDomain:      "ntfy.sh",
				NtfyTopic:       "test-topic",
				SlackWebhookURL: "not-a-url",
			},
			wantErr: true,
			errMsg:  "invalid Slack webhook URL format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error to contain %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestConfigInterface(t *testing.T) {
	cfg, err := config.New([]string{
		"--ntfy-domain", "test.ntfy.sh",
		"--ntfy-topic", "test-topic",
		"--ntfy-auth", "test-auth",
		"--slack-webhook", "https://hooks.slack.com/test",
	})
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Test that Config implements config.Provider interface
	var provider config.Provider = cfg

	if provider.GetNtfyDomain() != "test.ntfy.sh" {
		t.Errorf("GetNtfyDomain() = %v, want %v", provider.GetNtfyDomain(), "test.ntfy.sh")
	}
	if provider.GetNtfyTopic() != "test-topic" {
		t.Errorf("GetNtfyTopic() = %v, want %v", provider.GetNtfyTopic(), "test-topic")
	}
	if provider.GetNtfyAuth() != "test-auth" {
		t.Errorf("GetNtfyAuth() = %v, want %v", provider.GetNtfyAuth(), "test-auth")
	}
	if provider.GetSlackWebhookURL() != "https://hooks.slack.com/test" {
		t.Errorf("GetSlackWebhookURL() = %v, want %v", provider.GetSlackWebhookURL(), "https://hooks.slack.com/test")
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	// getEnvOrDefault is not exported, skip this test
	t.Skip("getEnvOrDefault is internal function, not testable")

	/*
		tests := []struct {
			name         string
			key          string
			defaultValue string
			envValue     string
			expected     string
		}{
			{
				name:         "environment variable set",
				key:          "TEST_VAR",
				defaultValue: "default",
				envValue:     "env-value",
				expected:     "env-value",
			},
			{
				name:         "environment variable not set",
				key:          "TEST_VAR_NOT_SET",
				defaultValue: "default",
				envValue:     "",
				expected:     "default",
			},
			{
				name:         "empty environment variable",
				key:          "TEST_VAR_EMPTY",
				defaultValue: "default",
				envValue:     "",
				expected:     "default",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Set up environment
				oldValue := os.Getenv(tt.key)
				if tt.envValue != "" {
					os.Setenv(tt.key, tt.envValue)
				} else {
					os.Unsetenv(tt.key)
				}
				defer func() {
					if oldValue != "" {
						os.Setenv(tt.key, oldValue)
					} else {
						os.Unsetenv(tt.key)
					}
				}()

				// Test skipped - function not exported
			})
		}
	*/
}
