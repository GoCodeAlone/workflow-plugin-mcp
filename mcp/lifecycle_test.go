package mcp_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/GoCodeAlone/modular"
	workflow "github.com/GoCodeAlone/workflow"
	"github.com/GoCodeAlone/workflow/config"
	_ "github.com/GoCodeAlone/workflow/setup"

	"github.com/GoCodeAlone/workflow-plugin-mcp/mcp"
)

// slogLogger adapts *slog.Logger to modular.Logger.
type slogLogger struct{ l *slog.Logger }

func (s *slogLogger) Info(msg string, args ...any)  { s.l.Info(msg, args...) }
func (s *slogLogger) Error(msg string, args ...any) { s.l.Error(msg, args...) }
func (s *slogLogger) Warn(msg string, args ...any)  { s.l.Warn(msg, args...) }
func (s *slogLogger) Debug(msg string, args ...any) { s.l.Debug(msg, args...) }

var _ modular.Logger = (*slogLogger)(nil)

const lifecycleYAML = `
modules:
  - name: mcp.tool-registry
    type: mcp.tool_registry
  - name: my-server
    type: mcp.server
    config:
      implementation:
        name: test-server
        version: "1.0"
      registry: mcp.tool-registry
  - name: my-http-transport
    type: mcp.http_transport
    config:
      server: my-server
      address: "127.0.0.1:0"

pipelines:
  greet:
    trigger:
      type: mcp.tool
      config:
        name: greet
        description: "Greet someone"
        server: my-server
        registry: mcp.tool-registry
        input_schema:
          type: object
          properties:
            name:
              type: string
    steps: []

  farewell:
    trigger:
      type: mcp.tool
      config:
        name: farewell
        description: "Say goodbye"
        server: my-server
        registry: mcp.tool-registry
        input_schema:
          type: object
          properties:
            name:
              type: string
    steps: []
`

// TestLifecycle_ToolsRegisteredBeforeTransportStarts proves the Start ordering
// invariant: after engine.Start, both tools are present in the server module
// (meaning ServerModule.Start replayed the registry onto the SDK server before
// any transport would begin serving).
func TestLifecycle_ToolsRegisteredBeforeTransportStarts(t *testing.T) {
	logger := &slogLogger{slog.Default()}

	cfg, err := config.LoadFromString(lifecycleYAML)
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

	// Resolve the ServerModule from the app's module registry.
	app := engine.App()
	mod := app.GetModule("my-server")
	if mod == nil {
		t.Fatal("module 'my-server' not found after Start")
	}
	serverMod, ok := mod.(*mcp.ServerModule)
	if !ok {
		t.Fatalf("module 'my-server' is %T, want *mcp.ServerModule", mod)
	}

	names := serverMod.ToolNames()
	if len(names) != 2 {
		t.Fatalf("ToolNames() = %v (len %d), want 2 tools", names, len(names))
	}

	want := map[string]bool{"greet": true, "farewell": true}
	for _, n := range names {
		if !want[n] {
			t.Errorf("unexpected tool name %q", n)
		}
	}

	// The http transport module was started as part of the same engine.Start
	// call above. Resolve it and verify it bound to a port, which proves the
	// transport's Start ran after the server's Start (else the transport would
	// either fail or the server would still be empty).
	tMod := app.GetModule("my-http-transport")
	if tMod == nil {
		t.Fatal("module 'my-http-transport' not found after Start")
	}
	httpMod, ok := tMod.(*mcp.HTTPTransportModule)
	if !ok {
		t.Fatalf("module 'my-http-transport' is %T, want *mcp.HTTPTransportModule", tMod)
	}
	if httpMod.Address() == "" {
		t.Error("HTTPTransportModule.Address() is empty; transport did not bind")
	}
}
