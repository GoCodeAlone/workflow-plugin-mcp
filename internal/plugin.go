package internal

import (
	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

// Manifest returns the plugin metadata used by the workflow engine for
// discovery and capability negotiation.
var Manifest = sdk.PluginManifest{
	Name:        "workflow-plugin-mcp",
	Version:     "0.1.0",
	Description: "MCP (Model Context Protocol) plugin for the workflow engine",
	Author:      "GoCodeAlone",
}

type plugin struct{}

// NewPlugin creates a new plugin instance.
func NewPlugin() sdk.PluginProvider {
	return &plugin{}
}

func (p *plugin) Manifest() sdk.PluginManifest {
	return Manifest
}
