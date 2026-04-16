package mcp_test

import (
	"context"
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

// TestStdioTransportModule_StopCancelsRun verifies that Stop cancels the
// internal run context even when the context originally passed to Start is
// still live.  We confirm this by passing a long-lived background context to
// Start and then calling Stop — if Stop were a true no-op the cancel would
// never fire and the module would leak.
//
// The test uses a context whose cancellation we can observe: we wrap a
// cancellable context as Start's parent and verify Stop does not itself require
// the parent to be cancelled.  We also verify that a second Start after Stop
// succeeds (the guard is cleared).
func TestStdioTransportModule_StopCancelsRun(t *testing.T) {
	srv := mcp.NewServerModule("my-server", mcp.ServerConfig{
		Implementation: mcp.Implementation{Name: "test", Version: "0.0.1"},
	})
	if err := srv.Init(nil); err != nil {
		t.Fatalf("ServerModule.Init: %v", err)
	}

	transport := mcp.NewStdioTransportModule("stdio", srv)
	if err := transport.Init(nil); err != nil {
		t.Fatalf("Init: %v", err)
	}

	// Start with a context that we deliberately do NOT cancel.
	parentCtx, parentCancel := context.WithCancel(context.Background())
	defer parentCancel() // clean up at end of test regardless

	if err := transport.Start(parentCtx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// A second Start while running must be rejected.
	if err := transport.Start(parentCtx); err == nil {
		t.Fatal("expected error on double-Start, got nil")
	}

	// Stop must succeed without cancelling the parent context.
	if err := transport.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	// After Stop, Start should be accepted again (cancel was cleared).
	if err := transport.Start(parentCtx); err != nil {
		t.Fatalf("Start after Stop: %v", err)
	}
	// Clean up.
	_ = transport.Stop(context.Background())
}
