package main

// This file is package main, not main_test: it exercises setupLogging
// directly, which is unexported and so unreachable from outside the
// package. Before tests were colocated with their source packages, this
// was a dead skip stub with no way to actually run it -- see this file's
// git history (tests/unit/app_test.go and tests/unit/main_test.go) for
// what it looked like before.

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestSetupLogging(t *testing.T) {
	tests := []struct {
		name      string
		logLevel  string
		wantLevel slog.Level
		wantWarn  bool
	}{
		{name: "debug level", logLevel: "debug", wantLevel: slog.LevelDebug},
		{name: "info level", logLevel: "info", wantLevel: slog.LevelInfo},
		{name: "warn level", logLevel: "warn", wantLevel: slog.LevelWarn},
		{name: "error level", logLevel: "error", wantLevel: slog.LevelError},
		{name: "empty level defaults to info", logLevel: "", wantLevel: slog.LevelInfo},
		{name: "invalid level defaults to info and warns", logLevel: "trace", wantLevel: slog.LevelInfo, wantWarn: true},
		{name: "case-sensitive: DEBUG is not debug, defaults to info and warns", logLevel: "DEBUG", wantLevel: slog.LevelInfo, wantWarn: true},
	}

	// setupLogging mutates process-wide slog state (the default logger and
	// the level SetLogLoggerLevel manages), so preserve and restore both
	// around the whole test.
	originalDefault := slog.Default()
	originalLevel := slog.SetLogLoggerLevel(slog.LevelInfo)
	defer func() {
		slog.SetDefault(originalDefault)
		slog.SetLogLoggerLevel(originalLevel)
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			// A handler built with nil options tracks the shared level
			// SetLogLoggerLevel manages, which is what lets setupLogging's
			// call to it actually take effect for this handler.
			slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))

			setupLogging(tt.logLevel)

			// SetLogLoggerLevel returns the previously configured level, so
			// calling it again reads back what setupLogging just set (as a
			// side effect it also resets the level, which is fine since the
			// whole test restores the original level and handler on exit).
			gotLevel := slog.SetLogLoggerLevel(tt.wantLevel)
			if gotLevel != tt.wantLevel {
				t.Errorf("setupLogging(%q) set level %v, want %v", tt.logLevel, gotLevel, tt.wantLevel)
			}

			gotWarn := strings.Contains(buf.String(), "invalid LOG_LEVEL")
			if gotWarn != tt.wantWarn {
				t.Errorf("setupLogging(%q): warned = %v, want %v (log output: %q)", tt.logLevel, gotWarn, tt.wantWarn, buf.String())
			}
		})
	}
}
