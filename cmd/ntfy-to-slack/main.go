package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

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

	// Run until asked to stop (e.g. `docker stop`, which sends SIGTERM).
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Run application
	if err := application.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("application error", "err", err)
		os.Exit(1)
	}
}

// setupLogging configures the application logging. An unrecognized level
// defaults to info but is reported, rather than silently accepted -- a
// typo'd LOG_LEVEL should be visible, not just quietly ignored.
func setupLogging(logLevel string) {
	switch logLevel {
	case "", "info":
		slog.SetLogLoggerLevel(slog.LevelInfo)
	case "debug":
		slog.SetLogLoggerLevel(slog.LevelDebug)
	case "warn":
		slog.SetLogLoggerLevel(slog.LevelWarn)
	case "error":
		slog.SetLogLoggerLevel(slog.LevelError)
	default:
		slog.SetLogLoggerLevel(slog.LevelInfo)
		slog.Warn("invalid LOG_LEVEL, defaulting to info", "log_level", logLevel)
	}
}
