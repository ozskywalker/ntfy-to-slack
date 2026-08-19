package main

// This file is package main, not main_test: it exercises startHealthServer
// directly, which is unexported.

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestStartHealthServer_ServesHandlerAndReturnsBoundAddr(t *testing.T) {
	var handlerCalled bool
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// "127.0.0.1:0" asks the OS for an ephemeral free port, so this doesn't
	// collide with anything else bound on the test runner.
	server, boundAddr, err := startHealthServer("127.0.0.1:0", handler)
	if err != nil {
		t.Fatalf("startHealthServer() error = %v", err)
	}
	t.Cleanup(func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	})

	if boundAddr == "" {
		t.Fatal("startHealthServer() returned an empty bound address")
	}

	resp, err := http.Get("http://" + boundAddr + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if !handlerCalled {
		t.Error("the handler passed to startHealthServer was never invoked")
	}
}

func TestStartHealthServer_BindFailureReturnsError(t *testing.T) {
	// Reserve a port so the address is genuinely already in use, then try
	// to start a second server on the exact same address.
	reserved, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to reserve a port for the test: %v", err)
	}
	defer reserved.Close()

	server, boundAddr, err := startHealthServer(reserved.Addr().String(), http.NotFoundHandler())
	if err == nil {
		server.Close()
		t.Fatalf("expected an error binding to an address already in use, got boundAddr=%q", boundAddr)
	}
	if server != nil {
		t.Error("expected a nil server on bind failure")
	}
}
