package unit_test

import (
	"testing"

	"github.com/ozskywalker/ntfy-to-slack/internal/config"
)


func TestNewConfig_PostProcessorFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected map[string]string
	}{
		{
			name: "webhook flag",
			args: []string{"--post-process-webhook", "https://example.com/webhook"},
			expected: map[string]string{
				"webhook": "https://example.com/webhook",
			},
		},
		{
			name: "template file flag",
			args: []string{"--post-process-template-file", "/path/to/template.tmpl"},
			expected: map[string]string{
				"templateFile": "/path/to/template.tmpl",
			},
		},
		{
			name: "inline template flag",
			args: []string{"--post-process-template", "{{.Title}}: {{.Message}}"},
			expected: map[string]string{
				"template": "{{.Title}}: {{.Message}}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := config.New(tt.args)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			
			if webhook, ok := tt.expected["webhook"]; ok {
				if config.PostProcessWebhook != webhook {
					t.Errorf("Expected webhook %q, got %q", webhook, config.PostProcessWebhook)
				}
			}
			
			if templateFile, ok := tt.expected["templateFile"]; ok {
				if config.PostProcessTemplateFile != templateFile {
					t.Errorf("Expected template file %q, got %q", templateFile, config.PostProcessTemplateFile)
				}
			}
			
			if template, ok := tt.expected["template"]; ok {
				if config.PostProcessTemplate != template {
					t.Errorf("Expected template %q, got %q", template, config.PostProcessTemplate)
				}
			}
		})
	}
}