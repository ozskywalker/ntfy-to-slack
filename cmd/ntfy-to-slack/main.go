package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/ozskywalker/ntfy-to-slack/internal/app"
	"github.com/ozskywalker/ntfy-to-slack/internal/config"
	"github.com/ozskywalker/ntfy-to-slack/internal/version"
)

func main() {
	// Parse configuration
	cfg, err := config.New(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	// Handle version flag
	if cfg.ShowVersion {
		fmt.Println(version.Get().Detailed())
		os.Exit(0)
	}

	// Handle help flag
	if cfg.ShowHelp {
		app.New(cfg, version.Get().String()).PrintHelp()
		os.Exit(1)
	}

	// Setup logging
	setupLogging(cfg.LogLevel)

	// Validate configuration before constructing the application from it, so
	// e.g. a malformed post-process template is rejected at startup instead
	// of being silently swapped for default formatting once running.
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	// Create application
	application := app.New(cfg, version.Get().String())

	// Run application
	if err := application.Run(); err != nil {
		slog.Error("application error", "err", err)
		os.Exit(1)
	}
}

// setupLogging configures the application logging
func setupLogging(logLevel string) {
	var level slog.Level
	switch logLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	slog.SetLogLoggerLevel(level)
}
