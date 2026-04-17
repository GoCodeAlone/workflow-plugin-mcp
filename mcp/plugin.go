package mcp

import (
	"github.com/GoCodeAlone/workflow/plugin"
	"github.com/GoCodeAlone/workflow/schema"
)

// Compile-time assertion.
var _ plugin.EnginePlugin = (*MCPPlugin)(nil)

// MCPPlugin registers the four MCP module types and the mcp.tool trigger
// with the workflow engine.
type MCPPlugin struct {
	plugin.BaseEnginePlugin
}

// New returns an MCPPlugin ready to be passed to engine.LoadPlugin.
func New() *MCPPlugin {
	return &MCPPlugin{
		BaseEnginePlugin: plugin.BaseEnginePlugin{
			BaseNativePlugin: plugin.BaseNativePlugin{
				PluginName:        "workflow-plugin-mcp",
				PluginVersion:     "0.1.0",
				PluginDescription: "MCP (Model Context Protocol) plugin",
			},
			Manifest: plugin.PluginManifest{
				Name:         "workflow-plugin-mcp",
				Version:      "0.1.0",
				Author:       "GoCodeAlone",
				Description:  "MCP server, transports, and tool trigger",
				Tier:         plugin.TierCommunity,
				ModuleTypes:  []string{"mcp.server", "mcp.stdio_transport", "mcp.http_transport", "mcp.tool_registry"},
				TriggerTypes: []string{"mcp.tool"},
			},
		},
	}
}

// ModuleFactories returns factories for the four MCP module types.
func (p *MCPPlugin) ModuleFactories() map[string]plugin.ModuleFactory {
	return map[string]plugin.ModuleFactory{
		"mcp.server":          serverModuleFactory,
		"mcp.stdio_transport": stdioTransportFactory,
		"mcp.http_transport":  httpTransportFactory,
		"mcp.tool_registry":   toolRegistryFactory,
	}
}

// TriggerFactories returns a factory for the mcp.tool trigger type.
func (p *MCPPlugin) TriggerFactories() map[string]plugin.TriggerFactory {
	return map[string]plugin.TriggerFactory{
		"mcp.tool": func() any { return NewToolTrigger() },
	}
}

// ModuleSchemas returns UI schema definitions for the four MCP module types.
func (p *MCPPlugin) ModuleSchemas() []*schema.ModuleSchema {
	return moduleSchemas()
}
