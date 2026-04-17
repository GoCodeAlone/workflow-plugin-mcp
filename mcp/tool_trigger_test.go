package mcp_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/GoCodeAlone/modular"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/GoCodeAlone/workflow-plugin-mcp/mcp"
)

var errDispatch = errors.New("pipeline dispatch failed")

// -----------------------------------------------------------------------------
// Test helpers / fakes
// -----------------------------------------------------------------------------

// fakeApp is a minimal modular.Application whose only functional method is
// SvcRegistry(). All other methods are no-op stubs to satisfy the interface.
type fakeApp struct {
	registry modular.ServiceRegistry
}

func newFakeApp() *fakeApp {
	return &fakeApp{registry: make(modular.ServiceRegistry)}
}

// SvcRegistry is the only method ToolTrigger.Configure actually calls.
func (a *fakeApp) SvcRegistry() modular.ServiceRegistry { return a.registry }

// --- modular.Application stubs ---

func (a *fakeApp) ConfigProvider() modular.ConfigProvider                                        { return nil }
func (a *fakeApp) RegisterModule(_ modular.Module)                                               {}
func (a *fakeApp) RegisterConfigSection(_ string, _ modular.ConfigProvider)                      {}
func (a *fakeApp) ConfigSections() map[string]modular.ConfigProvider                             { return nil }
func (a *fakeApp) GetConfigSection(_ string) (modular.ConfigProvider, error)                     { return nil, nil }
func (a *fakeApp) RegisterService(name string, svc any) error                                    { a.registry[name] = svc; return nil }
func (a *fakeApp) GetService(_ string, _ any) error                                              { return nil }
func (a *fakeApp) Init() error                                                                    { return nil }
func (a *fakeApp) Start() error                                                                   { return nil }
func (a *fakeApp) Stop() error                                                                    { return nil }
func (a *fakeApp) Run() error                                                                     { return nil }
func (a *fakeApp) Logger() modular.Logger                                                         { return nil }
func (a *fakeApp) SetLogger(_ modular.Logger)                                                     {}
func (a *fakeApp) SetVerboseConfig(_ bool)                                                        {}
func (a *fakeApp) IsVerboseConfig() bool                                                          { return false }
func (a *fakeApp) GetServicesByModule(_ string) []string                                          { return nil }
func (a *fakeApp) GetServiceEntry(_ string) (*modular.ServiceRegistryEntry, bool)                 { return nil, false }
func (a *fakeApp) GetServicesByInterface(_ reflect.Type) []*modular.ServiceRegistryEntry         { return nil }
func (a *fakeApp) StartTime() time.Time                                                           { return time.Time{} }
func (a *fakeApp) GetModule(_ string) modular.Module                                              { return nil }
func (a *fakeApp) GetAllModules() map[string]modular.Module                                      { return nil }
func (a *fakeApp) OnConfigLoaded(_ func(modular.Application) error)                              {}

// Compile-time assertion: fakeApp must satisfy modular.Application.
var _ modular.Application = (*fakeApp)(nil)

// -----------------------------------------------------------------------------

// mockExecutor records calls and returns a fixed response.
type mockExecutor struct {
	callCount int
	response  map[string]any
	err       error
}

func (m *mockExecutor) ExecutePipeline(_ context.Context, _ string, _ map[string]any) (map[string]any, error) {
	m.callCount++
	return m.response, m.err
}

// objectSchema is a minimal JSON-schema with "type":"object".
var objectSchema = map[string]any{
	"type": "object",
}

// nameRequiredSchema is a JSON-schema that requires a "name" string property.
var nameRequiredSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"name": map[string]any{"type": "string"},
	},
	"required": []any{"name"},
}

// baseTriggerConfig builds a minimal trigger config map; caller may override
// any key via the overrides map.
func baseTriggerConfig(overrides map[string]any) map[string]any {
	cfg := map[string]any{
		"workflowType": "pipeline:test-pipeline",
		"server":       "mcp.server",
		"name":         "test-tool",
		"description":  "A test tool",
	}
	for k, v := range overrides {
		cfg[k] = v
	}
	return cfg
}

// setupStandardApp wires a fakeApp with a ToolRegistry and a mockExecutor.
// The registry is stored under "mcp.tool-registry"; the executor is stored
// under "pipeline.executor".
func setupStandardApp(t *testing.T, exec *mockExecutor) (*fakeApp, *mcp.ToolRegistry) {
	t.Helper()
	app := newFakeApp()
	reg := mcp.NewToolRegistry("mcp.tool-registry")
	app.registry["mcp.tool-registry"] = reg
	app.registry["pipeline.executor"] = exec
	return app, reg
}

// -----------------------------------------------------------------------------
// Tests
// -----------------------------------------------------------------------------

// TestToolTrigger_ConfigureRegistersWithRegistry verifies that Configure adds
// one tool entry to the registry with the expected name and a non-nil handler.
func TestToolTrigger_ConfigureRegistersWithRegistry(t *testing.T) {
	exec := &mockExecutor{response: map[string]any{"ok": true}}
	app, reg := setupStandardApp(t, exec)

	trigger := mcp.NewToolTrigger()
	cfg := baseTriggerConfig(map[string]any{"input_schema": objectSchema})

	if err := trigger.Configure(app, cfg); err != nil {
		t.Fatalf("Configure: %v", err)
	}

	tools := reg.All()
	if len(tools) != 1 {
		t.Fatalf("expected 1 registered tool, got %d", len(tools))
	}
	if tools[0].Tool.Name != "test-tool" {
		t.Errorf("tool name = %q, want %q", tools[0].Tool.Name, "test-tool")
	}
	if tools[0].Handler == nil {
		t.Error("expected non-nil Handler")
	}
}

// TestToolTrigger_ValidatesInputAgainstSchema verifies that a handler call with
// an argument map that violates the schema returns IsError:true and does NOT
// invoke the executor.
func TestToolTrigger_ValidatesInputAgainstSchema(t *testing.T) {
	exec := &mockExecutor{}
	app, reg := setupStandardApp(t, exec)

	trigger := mcp.NewToolTrigger()
	cfg := baseTriggerConfig(map[string]any{"input_schema": nameRequiredSchema})
	if err := trigger.Configure(app, cfg); err != nil {
		t.Fatalf("Configure: %v", err)
	}

	handler := reg.All()[0].Handler

	// Call with empty object — missing required "name".
	req := &mcpsdk.CallToolRequest{}
	req.Params = &mcpsdk.CallToolParamsRaw{
		Name:      "test-tool",
		Arguments: json.RawMessage(`{}`),
	}
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected handler error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true for invalid input")
	}
	if len(result.Content) == 0 {
		t.Fatal("expected at least one content item")
	}
	tc, ok := result.Content[0].(*mcpsdk.TextContent)
	if !ok {
		t.Fatalf("expected *TextContent, got %T", result.Content[0])
	}
	if tc.Text == "" {
		t.Error("expected non-empty error text")
	}
	// The executor must NOT have been called.
	if exec.callCount != 0 {
		t.Errorf("executor called %d times; want 0", exec.callCount)
	}
}

// TestToolTrigger_DispatchesToPipeline verifies the happy path: valid input
// flows to the executor and the result is reflected in both Content and
// StructuredContent.
func TestToolTrigger_DispatchesToPipeline(t *testing.T) {
	pipelineOutput := map[string]any{"greeting": "hi bob"}
	exec := &mockExecutor{response: pipelineOutput}
	app, reg := setupStandardApp(t, exec)

	trigger := mcp.NewToolTrigger()
	cfg := baseTriggerConfig(map[string]any{"input_schema": nameRequiredSchema})
	if err := trigger.Configure(app, cfg); err != nil {
		t.Fatalf("Configure: %v", err)
	}

	handler := reg.All()[0].Handler

	req := &mcpsdk.CallToolRequest{}
	req.Params = &mcpsdk.CallToolParamsRaw{
		Name:      "test-tool",
		Arguments: json.RawMessage(`{"name":"bob"}`),
	}
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected handler error: %v", err)
	}
	if result.IsError {
		t.Errorf("IsError=true; content: %v", result.Content)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected at least one content item")
	}
	tc, ok := result.Content[0].(*mcpsdk.TextContent)
	if !ok {
		t.Fatalf("expected *TextContent, got %T", result.Content[0])
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(tc.Text), &got); err != nil {
		t.Fatalf("content text is not valid JSON: %v — text: %s", err, tc.Text)
	}
	if got["greeting"] != "hi bob" {
		t.Errorf("content 'greeting' = %v, want %q", got["greeting"], "hi bob")
	}

	sc, ok := result.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("StructuredContent type %T, want map[string]any", result.StructuredContent)
	}
	if sc["greeting"] != "hi bob" {
		t.Errorf("StructuredContent 'greeting' = %v, want %q", sc["greeting"], "hi bob")
	}

	if exec.callCount != 1 {
		t.Errorf("executor called %d times; want 1", exec.callCount)
	}
}

// TestToolTrigger_ResolvesSchemaRefFromConfigDir verifies that a $ref in
// input_schema is resolved relative to config_dir.
func TestToolTrigger_ResolvesSchemaRefFromConfigDir(t *testing.T) {
	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "greet.json")
	schemaJSON, _ := json.Marshal(nameRequiredSchema)
	if err := os.WriteFile(schemaPath, schemaJSON, 0o644); err != nil {
		t.Fatalf("write schema file: %v", err)
	}

	exec := &mockExecutor{response: map[string]any{}}
	app, reg := setupStandardApp(t, exec)

	trigger := mcp.NewToolTrigger()
	cfg := baseTriggerConfig(map[string]any{
		"input_schema": map[string]any{"$ref": "./greet.json"},
		"config_dir":   dir,
	})
	if err := trigger.Configure(app, cfg); err != nil {
		t.Fatalf("Configure with $ref: %v", err)
	}

	tools := reg.All()
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool after $ref configure, got %d", len(tools))
	}

	// The loaded schema should have "type":"object".
	is, ok := tools[0].Tool.InputSchema.(map[string]any)
	if !ok {
		t.Fatalf("InputSchema is %T, want map[string]any", tools[0].Tool.InputSchema)
	}
	if is["type"] != "object" {
		t.Errorf("loaded schema type = %v, want 'object'", is["type"])
	}
}

// TestToolTrigger_DispatchFailureReturnsIsError verifies that a pipeline
// dispatch error surfaces as IsError:true rather than a Go error.
func TestToolTrigger_DispatchFailureReturnsIsError(t *testing.T) {
	exec := &mockExecutor{err: errDispatch}
	app, reg := setupStandardApp(t, exec)

	trigger := mcp.NewToolTrigger()
	cfg := baseTriggerConfig(map[string]any{"input_schema": objectSchema})
	if err := trigger.Configure(app, cfg); err != nil {
		t.Fatalf("Configure: %v", err)
	}

	handler := reg.All()[0].Handler
	req := &mcpsdk.CallToolRequest{}
	req.Params = &mcpsdk.CallToolParamsRaw{
		Name:      "test-tool",
		Arguments: json.RawMessage(`{}`),
	}
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected handler error: %v (should be wrapped into IsError)", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true when pipeline dispatch fails")
	}
	if exec.callCount != 1 {
		t.Errorf("executor called %d times; want 1", exec.callCount)
	}
}

// TestToolTrigger_RejectsDuplicateToolName verifies that a second Configure
// with the same tool name returns an error rather than silently overwriting.
func TestToolTrigger_RejectsDuplicateToolName(t *testing.T) {
	exec := &mockExecutor{}
	app, _ := setupStandardApp(t, exec)

	trigger := mcp.NewToolTrigger()
	cfg := baseTriggerConfig(map[string]any{"input_schema": objectSchema})
	if err := trigger.Configure(app, cfg); err != nil {
		t.Fatalf("first Configure: %v", err)
	}
	// Second configure — same tool name, different pipeline.
	cfg2 := baseTriggerConfig(map[string]any{
		"input_schema": objectSchema,
		"workflowType": "pipeline:other-pipeline",
	})
	if err := trigger.Configure(app, cfg2); err == nil {
		t.Fatal("expected error for duplicate tool name, got nil")
	}
}

// TestToolTrigger_RejectsNonObjectTopLevelSchema verifies that Configure
// returns an error when the input schema top-level type is not "object".
func TestToolTrigger_RejectsNonObjectTopLevelSchema(t *testing.T) {
	exec := &mockExecutor{}
	app, _ := setupStandardApp(t, exec)

	trigger := mcp.NewToolTrigger()
	cfg := baseTriggerConfig(map[string]any{
		"input_schema": map[string]any{"type": "array"},
	})
	if err := trigger.Configure(app, cfg); err == nil {
		t.Fatal("expected error for non-object top-level schema, got nil")
	}
}

// TestToolTrigger_RejectsWhenServerOrRegistryOrExecutorMissing tests that
// Configure returns an error when required dependencies are absent.
func TestToolTrigger_RejectsWhenServerOrRegistryOrExecutorMissing(t *testing.T) {
	t.Run("missing registry", func(t *testing.T) {
		app := newFakeApp()
		// No registry in the service map; executor present.
		app.registry["pipeline.executor"] = &mockExecutor{}

		trigger := mcp.NewToolTrigger()
		cfg := baseTriggerConfig(map[string]any{"input_schema": objectSchema})
		if err := trigger.Configure(app, cfg); err == nil {
			t.Fatal("expected error for missing registry, got nil")
		}
	})

	t.Run("missing executor", func(t *testing.T) {
		app := newFakeApp()
		// Registry present, but no executor.
		app.registry["mcp.tool-registry"] = mcp.NewToolRegistry("mcp.tool-registry")

		trigger := mcp.NewToolTrigger()
		cfg := baseTriggerConfig(map[string]any{"input_schema": objectSchema})
		if err := trigger.Configure(app, cfg); err == nil {
			t.Fatal("expected error for missing executor, got nil")
		}
	})

	t.Run("missing workflowType", func(t *testing.T) {
		exec := &mockExecutor{}
		app, _ := setupStandardApp(t, exec)

		trigger := mcp.NewToolTrigger()
		cfg := baseTriggerConfig(nil)
		delete(cfg, "workflowType")
		if err := trigger.Configure(app, cfg); err == nil {
			t.Fatal("expected error for missing workflowType, got nil")
		}
	})

	t.Run("malformed workflowType", func(t *testing.T) {
		exec := &mockExecutor{}
		app, _ := setupStandardApp(t, exec)

		trigger := mcp.NewToolTrigger()
		cfg := baseTriggerConfig(map[string]any{"workflowType": "not-a-pipeline"})
		if err := trigger.Configure(app, cfg); err == nil {
			t.Fatal("expected error for workflowType without 'pipeline:' prefix, got nil")
		}
	})

	t.Run("server name is advisory (no server lookup in Configure)", func(t *testing.T) {
		// The server name is stored but not looked up until Task 2.5 wiring.
		// Configure must succeed even when the server module is absent from
		// the registry, as long as registry and executor are present.
		exec := &mockExecutor{}
		app, _ := setupStandardApp(t, exec)

		trigger := mcp.NewToolTrigger()
		cfg := baseTriggerConfig(map[string]any{
			"input_schema": objectSchema,
			"server":       "nonexistent-server", // not in registry — that's OK for now
		})
		if err := trigger.Configure(app, cfg); err != nil {
			t.Errorf("Configure failed with server absent from registry: %v (unexpected — server lookup deferred to Task 2.5)", err)
		}
	})
}
