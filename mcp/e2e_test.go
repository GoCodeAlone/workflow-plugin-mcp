package mcp_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	workflow "github.com/GoCodeAlone/workflow"
	"github.com/GoCodeAlone/workflow/config"
	_ "github.com/GoCodeAlone/workflow/setup"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/GoCodeAlone/workflow-plugin-mcp/mcp"
)

const e2eYAML = `
modules:
  - name: mcp.tool-registry
    type: mcp.tool_registry
  - name: e2e-server
    type: mcp.server
    config:
      implementation:
        name: e2e-test-server
        version: "0.1"
      registry: mcp.tool-registry

pipelines:
  greet:
    trigger:
      type: mcp.tool
      config:
        name: greet
        description: "Echo greeting"
        server: e2e-server
        registry: mcp.tool-registry
        input_schema:
          type: object
          properties:
            name:
              type: string
    steps: []
`

// TestE2E_ToolCallRoundTrip wires an in-memory MCP transport pair, starts the
// server, connects a client, and asserts that calling the "greet" tool returns
// a non-error result.
func TestE2E_ToolCallRoundTrip(t *testing.T) {
	logger := &slogLogger{slog.Default()}

	cfg, err := config.LoadFromString(e2eYAML)
	if err != nil {
		t.Fatalf("LoadFromString: %v", err)
	}

	engine, err := workflow.NewEngineBuilder().
		WithLogger(logger).
		WithAllDefaults().
		WithPlugin(mcp.New()).
		BuildFromConfig(cfg)
	if err != nil {
		t.Fatalf("BuildFromConfig: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := engine.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Retrieve the ServerModule and grab its underlying MCP server.
	app := engine.App()
	mod := app.GetModule("e2e-server")
	if mod == nil {
		t.Fatal("module 'e2e-server' not found")
	}
	serverMod, ok := mod.(*mcp.ServerModule)
	if !ok {
		t.Fatalf("module 'e2e-server' is %T, want *mcp.ServerModule", mod)
	}

	mcpServer := serverMod.Server()
	if mcpServer == nil {
		t.Fatal("ServerModule.Server() is nil after Start")
	}

	// Wire an in-memory transport pair.
	clientTr, serverTr := mcpsdk.NewInMemoryTransports()

	// Serve on the server transport in a goroutine.
	runCtx, runCancel := context.WithCancel(ctx)
	defer runCancel()
	go func() { _ = mcpServer.Run(runCtx, serverTr) }()

	// Connect a client.
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test-client", Version: "0"}, nil)
	session, err := client.Connect(ctx, clientTr, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	defer session.Close()

	// Call the "greet" tool.
	args, _ := json.Marshal(map[string]any{"name": "world"})
	result, err := session.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      "greet",
		Arguments: json.RawMessage(args),
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool call returned IsError=true; content: %v", result.Content)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected at least one content item in result")
	}
}
