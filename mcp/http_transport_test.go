package mcp_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/GoCodeAlone/workflow-plugin-mcp/mcp"
)

// newInitedServerModule is a test helper that returns a fully Init'd ServerModule.
func newInitedServerModule(t *testing.T) *mcp.ServerModule {
	t.Helper()
	srv := mcp.NewServerModule("test-server", mcp.ServerConfig{
		Implementation: mcp.Implementation{Name: "test", Version: "0.0.1"},
	})
	if err := srv.Init(nil); err != nil {
		t.Fatalf("ServerModule.Init: %v", err)
	}
	return srv
}

func TestHTTPTransportModule_InitValidatesConfig(t *testing.T) {
	t.Run("nil server", func(t *testing.T) {
		m := mcp.NewHTTPTransportModule("http", mcp.HTTPTransportConfig{Address: "127.0.0.1:9999"}, nil)
		if err := m.Init(nil); err == nil {
			t.Fatal("expected error for nil server, got nil")
		}
	})

	t.Run("uninitialised ServerModule (Server() is nil)", func(t *testing.T) {
		srv := mcp.NewServerModule("s", mcp.ServerConfig{
			Implementation: mcp.Implementation{Name: "test", Version: "0.0.1"},
		})
		m := mcp.NewHTTPTransportModule("http", mcp.HTTPTransportConfig{Address: "127.0.0.1:9999"}, srv)
		if err := m.Init(nil); err == nil {
			t.Fatal("expected error for uninitialised ServerModule, got nil")
		}
	})

	t.Run("empty address", func(t *testing.T) {
		srv := newInitedServerModule(t)
		m := mcp.NewHTTPTransportModule("http", mcp.HTTPTransportConfig{Address: ""}, srv)
		if err := m.Init(nil); err == nil {
			t.Fatal("expected error for empty address, got nil")
		}
	})

	t.Run("valid config", func(t *testing.T) {
		srv := newInitedServerModule(t)
		m := mcp.NewHTTPTransportModule("http", mcp.HTTPTransportConfig{Address: "127.0.0.1:9999"}, srv)
		if err := m.Init(nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

// TestHTTPTransportModule_StartAndStop uses address "127.0.0.1:0" so the OS
// picks a free ephemeral port.  Start binds synchronously and Address() returns
// the actual bound address — no TOCTOU probe-close-reuse window.
func TestHTTPTransportModule_StartAndStop(t *testing.T) {
	srv := newInitedServerModule(t)

	m := mcp.NewHTTPTransportModule("http", mcp.HTTPTransportConfig{Address: "127.0.0.1:0"}, srv)
	if err := m.Init(nil); err != nil {
		t.Fatalf("Init: %v", err)
	}

	if err := m.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	boundAddr := m.Address()
	if boundAddr == "" {
		t.Fatal("Address() returned empty string after Start")
	}

	// The listener is already bound; the server must be reachable immediately.
	// We still retry briefly to let the goroutine call Serve.
	url := "http://" + boundAddr + "/"
	deadline := time.Now().Add(time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		resp, err := http.Get(url) //nolint:noctx
		if err == nil {
			resp.Body.Close()
			lastErr = nil
			break
		}
		lastErr = err
		time.Sleep(5 * time.Millisecond)
	}
	if lastErr != nil {
		t.Fatalf("server did not become reachable within 1s: %v", lastErr)
	}

	// Graceful stop.
	stopCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := m.Stop(stopCtx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}

// TestHTTPTransportModule_StartBindError verifies that Start returns an error
// immediately when the address is already in use.
func TestHTTPTransportModule_StartBindError(t *testing.T) {
	// Start a first module to occupy the port.
	srv1 := newInitedServerModule(t)
	m1 := mcp.NewHTTPTransportModule("http1", mcp.HTTPTransportConfig{Address: "127.0.0.1:0"}, srv1)
	if err := m1.Init(nil); err != nil {
		t.Fatalf("m1.Init: %v", err)
	}
	if err := m1.Start(context.Background()); err != nil {
		t.Fatalf("m1.Start: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = m1.Stop(ctx)
	})

	// Try to bind the same address — must fail synchronously.
	srv2 := newInitedServerModule(t)
	m2 := mcp.NewHTTPTransportModule("http2", mcp.HTTPTransportConfig{Address: m1.Address()}, srv2)
	if err := m2.Init(nil); err != nil {
		t.Fatalf("m2.Init: %v", err)
	}
	if err := m2.Start(context.Background()); err == nil {
		t.Fatal("expected Start to return an error for already-bound address, got nil")
	}
}
