package internal_test

import (
	"testing"

	"github.com/GoCodeAlone/workflow-plugin-mcp/internal"
	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

func TestNewPlugin_ImplementsPluginProvider(t *testing.T) {
	var _ sdk.PluginProvider = internal.NewPlugin()
}

func TestManifest_HasRequiredFields(t *testing.T) {
	m := internal.Manifest
	if m.Name == "" {
		t.Error("manifest Name is empty")
	}
	if m.Version == "" {
		t.Error("manifest Version is empty")
	}
	if m.Description == "" {
		t.Error("manifest Description is empty")
	}
}

func TestNewPlugin_ImplementsModuleProvider(t *testing.T) {
	p := internal.NewPlugin()
	if _, ok := p.(sdk.ModuleProvider); !ok {
		t.Fatal("NewPlugin() does not implement sdk.ModuleProvider")
	}
}

func TestNewPlugin_ImplementsTriggerProvider(t *testing.T) {
	p := internal.NewPlugin()
	if _, ok := p.(sdk.TriggerProvider); !ok {
		t.Fatal("NewPlugin() does not implement sdk.TriggerProvider")
	}
}

func TestModuleTypes_ReturnsAllFour(t *testing.T) {
	mp := internal.NewPlugin().(sdk.ModuleProvider)
	want := map[string]bool{
		"mcp.server":          true,
		"mcp.stdio_transport": true,
		"mcp.http_transport":  true,
		"mcp.tool_registry":   true,
	}
	types := mp.ModuleTypes()
	if len(types) != len(want) {
		t.Fatalf("ModuleTypes() returned %d types, want %d: %v", len(types), len(want), types)
	}
	for _, typeName := range types {
		if !want[typeName] {
			t.Errorf("unexpected module type %q", typeName)
		}
	}
}

func TestCreateModule_ValidTypes(t *testing.T) {
	mp := internal.NewPlugin().(sdk.ModuleProvider)

	cases := []struct {
		typeName string
		cfg      map[string]any
	}{
		{"mcp.server", map[string]any{}},
		{"mcp.stdio_transport", map[string]any{}},
		{"mcp.http_transport", map[string]any{"address": ":9090"}},
		{"mcp.tool_registry", map[string]any{}},
	}

	for _, tc := range cases {
		t.Run(tc.typeName, func(t *testing.T) {
			inst, err := mp.CreateModule(tc.typeName, "test-"+tc.typeName, tc.cfg)
			if err != nil {
				t.Fatalf("CreateModule(%q) returned error: %v", tc.typeName, err)
			}
			if inst == nil {
				t.Fatalf("CreateModule(%q) returned nil instance", tc.typeName)
			}
			// Verify type assertion to sdk.ModuleInstance compiles and works.
			var _ sdk.ModuleInstance = inst
		})
	}
}

func TestCreateModule_UnknownType_ReturnsError(t *testing.T) {
	mp := internal.NewPlugin().(sdk.ModuleProvider)
	_, err := mp.CreateModule("mcp.unknown", "test", map[string]any{})
	if err == nil {
		t.Fatal("expected error for unknown module type, got nil")
	}
}

func TestTriggerTypes_ReturnsMCPTool(t *testing.T) {
	tp := internal.NewPlugin().(sdk.TriggerProvider)
	types := tp.TriggerTypes()
	if len(types) != 1 || types[0] != "mcp.tool" {
		t.Fatalf("TriggerTypes() = %v, want [mcp.tool]", types)
	}
}

func TestCreateTrigger_UnknownType_ReturnsError(t *testing.T) {
	tp := internal.NewPlugin().(sdk.TriggerProvider)
	_, err := tp.CreateTrigger("mcp.unknown", map[string]any{}, nil)
	if err == nil {
		t.Fatal("expected error for unknown trigger type, got nil")
	}
}

// TestCreateTrigger_MCPTool_MinimalConfig verifies CreateTrigger for mcp.tool
// succeeds with a minimal config that satisfies the trigger's Configure requirements.
func TestCreateTrigger_MCPTool_MinimalConfig(t *testing.T) {
	tp := internal.NewPlugin().(sdk.TriggerProvider)
	cfg := map[string]any{
		"workflowType": "pipeline:test-pipeline",
		"name":         "test-tool",
		"description":  "A test MCP tool",
	}
	inst, err := tp.CreateTrigger("mcp.tool", cfg, nil)
	if err != nil {
		t.Fatalf("CreateTrigger(mcp.tool) returned error: %v", err)
	}
	if inst == nil {
		t.Fatal("CreateTrigger(mcp.tool) returned nil instance")
	}
	var _ sdk.TriggerInstance = inst
}
