package mcp_test

import (
	"testing"

	"github.com/GoCodeAlone/workflow-plugin-mcp/mcp"
)

func TestStdioTransportModule_InitRequiresWiredServer(t *testing.T) {
	t.Run("nil server", func(t *testing.T) {
		transport := mcp.NewStdioTransportModule("stdio", nil)
		if err := transport.Init(nil); err == nil {
			t.Fatal("expected error when server is nil, got nil")
		}
	})

	t.Run("uninitialised ServerModule (Server() is nil)", func(t *testing.T) {
		// ServerModule constructed but Init never called, so server.Server() == nil.
		srv := mcp.NewServerModule("my-server", mcp.ServerConfig{
			Implementation: mcp.Implementation{Name: "test", Version: "0.0.1"},
		})
		transport := mcp.NewStdioTransportModule("stdio", srv)
		if err := transport.Init(nil); err == nil {
			t.Fatal("expected error when ServerModule has not been Init'd, got nil")
		}
	})
}

func TestStdioTransportModule_InitAcceptsInitializedServer(t *testing.T) {
	srv := mcp.NewServerModule("my-server", mcp.ServerConfig{
		Implementation: mcp.Implementation{Name: "test", Version: "0.0.1"},
	})
	if err := srv.Init(nil); err != nil {
		t.Fatalf("ServerModule.Init: %v", err)
	}

	transport := mcp.NewStdioTransportModule("stdio", srv)
	if err := transport.Init(nil); err != nil {
		t.Fatalf("StdioTransportModule.Init returned unexpected error: %v", err)
	}
}
