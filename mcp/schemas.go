package mcp

import "github.com/GoCodeAlone/workflow/schema"

// ModuleSchemas returns UI schema definitions for the four MCP module types.
func moduleSchemas() []*schema.ModuleSchema {
	return []*schema.ModuleSchema{
		{
			Type:         "mcp.server",
			Label:        "MCP Server",
			Category:     "mcp",
			Description:  "MCP server instance (Model Context Protocol)",
			ConfigFields: []schema.ConfigFieldDef{},
		},
		{
			Type:         "mcp.stdio_transport",
			Label:        "MCP Stdio Transport",
			Category:     "mcp",
			Description:  "Attaches a stdio transport to an MCP server",
			ConfigFields: []schema.ConfigFieldDef{},
		},
		{
			Type:         "mcp.http_transport",
			Label:        "MCP HTTP Transport",
			Category:     "mcp",
			Description:  "Attaches a Streamable-HTTP transport to an MCP server",
			ConfigFields: []schema.ConfigFieldDef{},
		},
		{
			Type:         "mcp.tool_registry",
			Label:        "MCP Tool Registry",
			Category:     "mcp",
			Description:  "Thread-safe registry of MCP tools contributed by pipeline triggers",
			ConfigFields: []schema.ConfigFieldDef{},
		},
	}
}
