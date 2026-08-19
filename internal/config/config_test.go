package config_test

import (
	"fmt"
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
		{
			name: "valid ntfy bearer token auth",
			config: &config.Config{
				NtfyDomain:      "ntfy.sh",
				NtfyTopic:       "test-topic",
				SlackWebhookURL: "https://hooks.slack.com/test",
				NtfyAuth:        "tk_sometoken",
			},
			wantErr: false,
		},
		{
			name: "valid ntfy username/password auth",
			config: &config.Config{
				NtfyDomain:      "ntfy.sh",
				NtfyTopic:       "test-topic",
				SlackWebhookURL: "https://hooks.slack.com/test",
				NtfyUsername:    "alice",
				NtfyPassword:    "hunter2",
			},
			wantErr: false,
		},
		{
			name: "ntfy auth token and username/password are mutually exclusive",
			config: &config.Config{
				NtfyDomain:      "ntfy.sh",
				NtfyTopic:       "test-topic",
				SlackWebhookURL: "https://hooks.slack.com/test",
				NtfyAuth:        "tk_sometoken",
				NtfyUsername:    "alice",
				NtfyPassword:    "hunter2",
			},
			wantErr: true,
			errMsg:  "only one ntfy authentication method",
		},
		{
			name: "ntfy username without password is rejected",
			config: &config.Config{
				NtfyDomain:      "ntfy.sh",
				NtfyTopic:       "test-topic",
				SlackWebhookURL: "https://hooks.slack.com/test",
				NtfyUsername:    "alice",
			},
			wantErr: true,
			errMsg:  "must both be specified together",
		},
		{
			name: "ntfy password without username is rejected",
			config: &config.Config{
				NtfyDomain:      "ntfy.sh",
				NtfyTopic:       "test-topic",
				SlackWebhookURL: "https://hooks.slack.com/test",
				NtfyPassword:    "hunter2",
			},
			wantErr: true,
			errMsg:  "must both be specified together",
		},
		{
			name: "health addr disabled by default is valid",
			config: &config.Config{
				NtfyDomain:      "ntfy.sh",
				NtfyTopic:       "test-topic",
				SlackWebhookURL: "https://hooks.slack.com/test",
			},
			wantErr: false,
		},
		{
			name: "valid health addr",
			config: &config.Config{
				NtfyDomain:      "ntfy.sh",
				NtfyTopic:       "test-topic",
				SlackWebhookURL: "https://hooks.slack.com/test",
				HealthAddr:      ":8080",
			},
			wantErr: false,
		},
		{
			name: "health addr missing leading colon is rejected",
			config: &config.Config{
				NtfyDomain:      "ntfy.sh",
				NtfyTopic:       "test-topic",
				SlackWebhookURL: "https://hooks.slack.com/test",
				HealthAddr:      "8080",
			},
			wantErr: true,
			errMsg:  "invalid health endpoint address",
		},
		{
			name: "valid inline template",
			config: &config.Config{
				NtfyDomain:          "ntfy.sh",
				NtfyTopic:           "test-topic",
				SlackWebhookURL:     "https://hooks.slack.com/test",
				PostProcessTemplate: "{{.Title}}: {{.Message}}",
			},
			wantErr: false,
		},
		{
			name: "invalid inline template syntax rejected at validation time",
			config: &config.Config{
				NtfyDomain:          "ntfy.sh",
				NtfyTopic:           "test-topic",
				SlackWebhookURL:     "https://hooks.slack.com/test",
				PostProcessTemplate: "{{.Title",
			},
			wantErr: true,
			errMsg:  "invalid post-process template",
		},
		{
			name: "non-existent template file rejected at validation time",
			config: &config.Config{
				NtfyDomain:              "ntfy.sh",
				NtfyTopic:               "test-topic",
				SlackWebhookURL:         "https://hooks.slack.com/test",
				PostProcessTemplateFile: "/nonexistent/path/template.tmpl",
			},
			wantErr: true,
			errMsg:  "invalid post-process template file",
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

// TestFlagUsage_ListsEveryFlag guards against the exact kind of drift that
// motivated generating this text in the first place: New() gained
// --ntfy-username/--ntfy-password without the (at the time hand-written)
// help text being updated to match. Every flag New()'s FlagSet defines must
// show up here, since they now share one definition (buildFlagSet).
func TestFlagUsage_ListsEveryFlag(t *testing.T) {
	usage := config.FlagUsage()

	for _, flagName := range []string{
		"-ntfy-domain",
		"-ntfy-topic",
		"-ntfy-auth",
		"-ntfy-username",
		"-ntfy-password",
		"-slack-webhook",
		"-post-process-webhook",
		"-post-process-template-file",
		"-post-process-template",
		"-webhook-timeout",
		"-webhook-retries",
		"-webhook-max-response-size",
		"-health-addr",
		"-v",
	} {
		if !strings.Contains(usage, flagName) {
			t.Errorf("FlagUsage() is missing %q:\n%s", flagName, usage)
		}
	}
}

// TestFlagUsage_DefaultsMatchNew guards the other half of the drift: not
// just that a flag is listed, but that the default FlagUsage shows for it
// is the same value New() would actually use when the flag isn't passed.
func TestFlagUsage_DefaultsMatchNew(t *testing.T) {
	oldTimeout, hadTimeout := os.LookupEnv("WEBHOOK_TIMEOUT_SECONDS")
	os.Unsetenv("WEBHOOK_TIMEOUT_SECONDS")
	defer func() {
		if hadTimeout {
			os.Setenv("WEBHOOK_TIMEOUT_SECONDS", oldTimeout)
		}
	}()

	cfg, err := config.New([]string{"--ntfy-topic", "t", "--slack-webhook", "https://hooks.slack.com/x"})
	if err != nil {
		t.Fatalf("config.New() error: %v", err)
	}

	usage := config.FlagUsage()

	if !strings.Contains(usage, `-ntfy-domain string`) || !strings.Contains(usage, `(default "`+cfg.GetNtfyDomain()+`")`) {
		t.Errorf("FlagUsage() default for -ntfy-domain doesn't match New()'s actual default %q:\n%s", cfg.GetNtfyDomain(), usage)
	}
	if !strings.Contains(usage, fmt.Sprintf("(default %d)", cfg.GetWebhookTimeoutSeconds())) {
		t.Errorf("FlagUsage() default for -webhook-timeout doesn't match New()'s actual default %d:\n%s", cfg.GetWebhookTimeoutSeconds(), usage)
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

// getEnvOrDefault and getEnvIntOrDefault are unexported; see
// internal_test.go (package config, not config_test) for coverage of them
// directly, now that colocating tests makes that possible.
