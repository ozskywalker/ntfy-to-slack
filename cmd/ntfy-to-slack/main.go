package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	if addr := cfg.GetHealthAddr(); addr != "" {
		healthServer := startHealthServer(addr, application.HealthHandler())
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := healthServer.Shutdown(shutdownCtx); err != nil {
				slog.Error("health endpoint shutdown error", "err", err)
			}
		}()
	}

	// Run application
	if err := application.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("application error", "err", err)
		os.Exit(1)
	}
}

// startHealthServer starts the /healthz liveness endpoint in the
// background and returns the server so the caller can shut it down.
func startHealthServer(addr string, handler http.Handler) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/healthz", handler)
	server := &http.Server{Addr: addr, Handler: mux}

	go func() {
		slog.Info("health endpoint listening", "addr", addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("health endpoint error", "err", err)
		}
	}()

	return server
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
