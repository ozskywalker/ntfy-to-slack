package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
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

	shutdownHealthServer, err := setupHealthServer(cfg.GetHealthAddr(), application.HealthHandler())
	if err != nil {
		// Config.Validate already checked addr parses as host:port; a
		// failure here means something else has that port (or the address
		// otherwise can't be bound) -- fail fast rather than silently
		// running without the health check the operator asked for, matching
		// how a bad post-process template is rejected at startup rather
		// than discovered later.
		slog.Error("failed to start health endpoint", "err", err)
		os.Exit(1)
	}
	defer shutdownHealthServer()

	// Run application
	if err := application.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("application error", "err", err)
		os.Exit(1)
	}
}

// setupHealthServer wires up the optional /healthz liveness endpoint: if
// addr is empty (the default -- health-addr wasn't set), it's a no-op that
// returns a no-op shutdown func so main can call it unconditionally. This
// split from startHealthServer exists so main's own logic (deciding
// whether to start the endpoint, and how to shut it down) is exercised by a
// test directly, instead of only ever running as part of main() itself,
// which os.Exit and signal handling make impractical to invoke from a test.
func setupHealthServer(addr string, handler http.Handler) (shutdown func(), err error) {
	if addr == "" {
		return func() {}, nil
	}

	server, _, err := startHealthServer(addr, handler)
	if err != nil {
		return nil, err
	}

	return func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("health endpoint shutdown error", "err", err)
		}
	}, nil
}

// startHealthServer starts the /healthz liveness endpoint in the
// background and returns the server (so the caller can shut it down) and
// the address it actually bound to. Binding happens synchronously, before
// this returns, so a bind failure (e.g. the port is already in use) is
// reported to the caller immediately rather than only ever logged from the
// background goroutine after the fact -- letting main fail fast on it the
// same way it already does for other startup-time configuration problems.
// The returned address matters for tests, which listen on "127.0.0.1:0" to
// get an OS-assigned port and need to know which one they got.
func startHealthServer(addr string, handler http.Handler) (server *http.Server, boundAddr string, err error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, "", err
	}

	mux := http.NewServeMux()
	mux.Handle("/healthz", handler)
	server = &http.Server{Handler: mux}

	go func() {
		slog.Info("health endpoint listening", "addr", listener.Addr().String())
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("health endpoint error", "err", err)
		}
	}()

	return server, listener.Addr().String(), nil
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
