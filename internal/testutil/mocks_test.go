package testutil

import "testing"

// TestMockConfigProvider_ImplementsProvider exercises every getter plus
// Validate through the config.Provider interface itself, the same way real
// callers (app.New, cmd/ntfy-to-slack) use a Provider -- so a typo'd field
// mapping (e.g. a getter returning the wrong struct field) would show up as
// a test failure here instead of only at the call site that happens to
// notice the wrong value.
func TestMockConfigProvider_ImplementsProvider(t *testing.T) {
	m := &MockConfigProvider{
		Domain:                   "ntfy.sh",
		Topic:                    "test-topic",
		Auth:                     "auth-token",
		Username:                 "alice",
		Password:                 "hunter2",
		WebhookURL:               "https://hooks.slack.com/test",
		PostProcessWebhook:       "https://example.com/webhook",
		PostProcessTemplate:      "{{.Title}}",
		PostProcessTemplateFile:  "/tmp/template.tmpl",
		WebhookTimeoutSeconds:    30,
		WebhookRetries:           3,
		WebhookMaxResponseSizeMB: 1,
		HealthAddr:               ":8080",
	}

	tests := []struct {
		name string
		got  any
		want any
	}{
		{"GetNtfyDomain", m.GetNtfyDomain(), m.Domain},
		{"GetNtfyTopic", m.GetNtfyTopic(), m.Topic},
		{"GetNtfyAuth", m.GetNtfyAuth(), m.Auth},
		{"GetNtfyUsername", m.GetNtfyUsername(), m.Username},
		{"GetNtfyPassword", m.GetNtfyPassword(), m.Password},
		{"GetSlackWebhookURL", m.GetSlackWebhookURL(), m.WebhookURL},
		{"GetPostProcessWebhook", m.GetPostProcessWebhook(), m.PostProcessWebhook},
		{"GetPostProcessTemplate", m.GetPostProcessTemplate(), m.PostProcessTemplate},
		{"GetPostProcessTemplateFile", m.GetPostProcessTemplateFile(), m.PostProcessTemplateFile},
		{"GetHealthAddr", m.GetHealthAddr(), m.HealthAddr},
		{"GetWebhookTimeoutSeconds", m.GetWebhookTimeoutSeconds(), m.WebhookTimeoutSeconds},
		{"GetWebhookRetries", m.GetWebhookRetries(), m.WebhookRetries},
		{"GetWebhookMaxResponseSizeMB", m.GetWebhookMaxResponseSizeMB(), m.WebhookMaxResponseSizeMB},
	}

	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s() = %v, want %v", tt.name, tt.got, tt.want)
		}
	}

	if err := m.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}
}
