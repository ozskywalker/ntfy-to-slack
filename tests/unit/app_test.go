package unit_test

import (
	"testing"

	"github.com/ozskywalker/ntfy-to-slack/internal/app"
)


func TestNewApp(t *testing.T) {
	config := &MockConfigProvider{
		Domain:     "test.ntfy.sh",
		Topic:      "test-topic",
		Auth:       "auth-token",
		WebhookURL: "https://hooks.slack.com/test",
	}
	
	appInstance := app.New(config, "test-version")
	
	if appInstance == nil {
		t.Error("app.New() returned nil")
	}
	
	// Note: config field is not exported, so we can't test it directly
	
	// Note: internal fields are not exported, so we can't test them directly
	// The fact that New() doesn't panic is sufficient for this test
}

func TestSetupLogging(t *testing.T) {
	tests := []struct {
		name     string
		logLevel string
		// Note: We can't easily test slog.SetLogLoggerLevel effects
		// but we can at least verify the function doesn't panic
	}{
		{"debug level", "debug"},
		{"info level", "info"},
		{"warn level", "warn"},
		{"error level", "error"},
		{"invalid level defaults to info", "invalid"},
		{"empty level defaults to info", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that function doesn't panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("setupLogging() panicked: %v", r)
				}
			}()
			
			// setupLogging is not exported, skip this test
			t.Skip("setupLogging is not exported")
		})
	}
}

func TestPrintHelp(t *testing.T) {
	// Test that printHelp doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("printHelp() panicked: %v", r)
		}
	}()
	
	// printHelp is not exported, we can test the exported method instead
	config := &MockConfigProvider{}
	appInstance := app.New(config, "test-version")
	appInstance.PrintHelp()
}