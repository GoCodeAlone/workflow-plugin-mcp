package mcp_test

import (
	"testing"

	"github.com/GoCodeAlone/workflow-plugin-mcp/mcp"
)

func TestServerModule_InitCreatesServer(t *testing.T) {
	cfg := mcp.ServerConfig{
		Implementation: mcp.Implementation{
			Name:    "test-server",
			Version: "1.0.0",
		},
	}
	mod := mcp.NewServerModule("my-mcp", cfg)

	if err := mod.Init(nil); err != nil {
		t.Fatalf("unexpected error from Init: %v", err)
	}

	if mod.Server() == nil {
		t.Fatal("expected Server() to be non-nil after Init")
	}

	if got := mod.Name(); got != "my-mcp" {
		t.Errorf("Name() = %q, want %q", got, "my-mcp")
	}
}

func TestServerModule_RequiresNonEmptyImplementation(t *testing.T) {
	cfg := mcp.ServerConfig{
		Implementation: mcp.Implementation{
			Name:    "",
			Version: "1.0.0",
		},
	}
	mod := mcp.NewServerModule("my-mcp", cfg)

	if err := mod.Init(nil); err == nil {
		t.Fatal("expected error when Implementation.Name is empty, got nil")
	}
}
