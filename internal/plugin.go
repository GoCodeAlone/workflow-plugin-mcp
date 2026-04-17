package internal

import (
	"fmt"

	"github.com/GoCodeAlone/modular"
	"github.com/GoCodeAlone/workflow-plugin-mcp/mcp"
	"github.com/GoCodeAlone/workflow/interfaces"
	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

// Manifest is the plugin metadata used by the workflow engine for discovery
// and capability negotiation.
var Manifest = sdk.PluginManifest{
	Name:        "workflow-plugin-mcp",
	Version:     "0.1.0",
	Description: "MCP (Model Context Protocol) plugin for the workflow engine",
	Author:      "GoCodeAlone",
}

// plugin implements sdk.PluginProvider, sdk.ModuleProvider, and
// sdk.TriggerProvider for the external gRPC plugin mode.
//
// A single modular.Application is created at construction and shared across
// all module and trigger instances created by this plugin. In the gRPC
// subprocess the registry is empty (no host services available) — this is the
// documented v0.1.0 limitation. Full cross-module wiring requires the
// in-process library mode (mcp.MCPPlugin).
type plugin struct {
	app modular.Application
}

// Compile-time assertions.
var _ sdk.PluginProvider = (*plugin)(nil)
var _ sdk.ModuleProvider = (*plugin)(nil)
var _ sdk.TriggerProvider = (*plugin)(nil)

// NewPlugin creates a new plugin instance with a modular.Application
// pre-seeded for the gRPC subprocess context.
//
// Two services are registered in the shared app at construction time:
//   - A default ToolRegistry (keyed as "mcp.tool-registry") so that
//     ToolTrigger.Configure can resolve the registry without the host.
//   - A no-op PipelineExecutor so that Configure can resolve the executor.
//     Actual pipeline dispatch is not available in gRPC mode (v0.1.0 limitation).
func NewPlugin() sdk.PluginProvider {
	app := modular.NewStdApplication(nil, nil)

	// Pre-seed a default ToolRegistry so triggers can register tools.
	defaultRegistry := mcp.NewToolRegistry("mcp.tool-registry")
	_ = app.RegisterService("mcp.tool-registry", defaultRegistry)

	// Pre-seed a no-op executor so Configure doesn't fail on the executor
	// lookup. Calls through this executor return a descriptive error.
	_ = app.RegisterService("_noop_pipeline_executor", interfaces.PipelineExecutor(noopPipelineExecutor{}))

	return &plugin{app: app}
}

// Manifest implements sdk.PluginProvider.
func (p *plugin) Manifest() sdk.PluginManifest {
	return Manifest
}

// ModuleTypes implements sdk.ModuleProvider.
func (p *plugin) ModuleTypes() []string {
	return []string{
		"mcp.server",
		"mcp.stdio_transport",
		"mcp.http_transport",
		"mcp.tool_registry",
	}
}

// CreateModule implements sdk.ModuleProvider. It parses the config, constructs
// the appropriate module, and wraps it in a moduleAdapter that bridges the SDK
// lifecycle to the modular framework.
func (p *plugin) CreateModule(typeName, name string, cfg map[string]any) (sdk.ModuleInstance, error) {
	switch typeName {
	case "mcp.server":
		sc, err := mcp.ParseServerConfig(cfg)
		if err != nil {
			return nil, err
		}
		return newModuleAdapter(p.app, mcp.NewServerModule(name, sc)), nil

	case "mcp.stdio_transport":
		serverName, err := mcp.ParseStdioTransportConfig(cfg)
		if err != nil {
			return nil, err
		}
		return newModuleAdapter(p.app, mcp.NewStdioTransportModuleByName(name, serverName)), nil

	case "mcp.http_transport":
		tc, serverName, err := mcp.ParseHTTPTransportConfig(cfg)
		if err != nil {
			return nil, err
		}
		return newModuleAdapter(p.app, mcp.NewHTTPTransportModuleByName(name, tc, serverName)), nil

	case "mcp.tool_registry":
		return newModuleAdapter(p.app, mcp.NewToolRegistry(name)), nil
	}

	return nil, fmt.Errorf("unknown module type %q", typeName)
}

// TriggerTypes implements sdk.TriggerProvider.
func (p *plugin) TriggerTypes() []string {
	return []string{"mcp.tool"}
}

// CreateTrigger implements sdk.TriggerProvider. It constructs a new ToolTrigger
// and eagerly calls Configure on it so that any config errors surface before
// Start is attempted.
//
// v0.1.0 note: the workflowType key must be present in cfg (the engine normally
// injects it as "pipeline:<name>"). If absent, Configure will return an error
// here. The TriggerCallback cb is stored but not wired in v0.1.0.
func (p *plugin) CreateTrigger(typeName string, cfg map[string]any, cb sdk.TriggerCallback) (sdk.TriggerInstance, error) {
	if typeName != "mcp.tool" {
		return nil, fmt.Errorf("unknown trigger type %q", typeName)
	}
	return newTriggerAdapter(p.app, mcp.NewToolTrigger(), cfg, cb)
}
