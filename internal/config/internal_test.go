package config

import (
	"os"
	"testing"
)

// This file is package config, not config_test: it exercises
// getEnvOrDefault and getEnvIntOrDefault directly, which are unexported and
// so unreachable from outside the package. Before tests were colocated with
// their source packages, these were dead skip stubs (see the git history of
// tests/unit/config_test.go) with no way to actually run them.

func TestGetEnvOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		setEnv       bool
		expected     string
	}{
		{
			name:         "environment variable set",
			key:          "TEST_GETENVORDEFAULT_SET",
			defaultValue: "default",
			envValue:     "env-value",
			setEnv:       true,
			expected:     "env-value",
		},
		{
			name:         "environment variable not set",
			key:          "TEST_GETENVORDEFAULT_UNSET",
			defaultValue: "default",
			setEnv:       false,
			expected:     "default",
		},
		{
			name:         "empty environment variable falls back to default",
			key:          "TEST_GETENVORDEFAULT_EMPTY",
			defaultValue: "default",
			envValue:     "",
			setEnv:       true,
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldValue, hadValue := os.LookupEnv(tt.key)
			if tt.setEnv {
				os.Setenv(tt.key, tt.envValue)
			} else {
				os.Unsetenv(tt.key)
			}
			defer func() {
				if hadValue {
					os.Setenv(tt.key, oldValue)
				} else {
					os.Unsetenv(tt.key)
				}
			}()

			if got := getEnvOrDefault(tt.key, tt.defaultValue); got != tt.expected {
				t.Errorf("getEnvOrDefault(%q, %q) = %q, want %q", tt.key, tt.defaultValue, got, tt.expected)
			}
		})
	}
}

func TestGetEnvIntOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue int
		envValue     string
		setEnv       bool
		expected     int
	}{
		{
			name:         "valid integer set",
			key:          "TEST_GETENVINTORDEFAULT_VALID",
			defaultValue: 30,
			envValue:     "45",
			setEnv:       true,
			expected:     45,
		},
		{
			name:         "environment variable not set",
			key:          "TEST_GETENVINTORDEFAULT_UNSET",
			defaultValue: 30,
			setEnv:       false,
			expected:     30,
		},
		{
			name:         "non-numeric value falls back to default",
			key:          "TEST_GETENVINTORDEFAULT_INVALID",
			defaultValue: 30,
			envValue:     "not-a-number",
			setEnv:       true,
			expected:     30,
		},
		{
			name:         "empty value falls back to default",
			key:          "TEST_GETENVINTORDEFAULT_EMPTY",
			defaultValue: 30,
			envValue:     "",
			setEnv:       true,
			expected:     30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldValue, hadValue := os.LookupEnv(tt.key)
			if tt.setEnv {
				os.Setenv(tt.key, tt.envValue)
			} else {
				os.Unsetenv(tt.key)
			}
			defer func() {
				if hadValue {
					os.Setenv(tt.key, oldValue)
				} else {
					os.Unsetenv(tt.key)
				}
			}()

			if got := getEnvIntOrDefault(tt.key, tt.defaultValue); got != tt.expected {
				t.Errorf("getEnvIntOrDefault(%q, %d) = %d, want %d", tt.key, tt.defaultValue, got, tt.expected)
			}
		})
	}
}
