package mcp_test

import (
	"context"
	"net"
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

// freeAddr probes the OS for a free TCP port by briefly binding, records the
// address, then closes the listener.  There is a small TOCTOU window but it
// is acceptable for tests on a loopback interface.
func freeAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close()
	return addr
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

func TestHTTPTransportModule_StartAndStop(t *testing.T) {
	addr := freeAddr(t)
	srv := newInitedServerModule(t)

	m := mcp.NewHTTPTransportModule("http", mcp.HTTPTransportConfig{Address: addr}, srv)
	if err := m.Init(nil); err != nil {
		t.Fatalf("Init: %v", err)
	}

	if err := m.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Poll until the server is reachable (allow up to 1 s for ListenAndServe to bind).
	url := "http://" + addr + "/"
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
		time.Sleep(10 * time.Millisecond)
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
