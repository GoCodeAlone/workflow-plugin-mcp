package mcp

import (
	"sync"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// RegisteredTool pairs an MCP Tool descriptor with its handler.
type RegisteredTool struct {
	Tool    *mcpsdk.Tool
	Handler mcpsdk.ToolHandler
}

// ToolRegistry is a thread-safe store of MCP tools to be registered with a
// ServerModule. It is intentionally a separate module so that plugins can
// contribute tools without importing ServerModule directly.
type ToolRegistry struct {
	name  string
	mu    sync.Mutex
	tools []RegisteredTool
}

// NewToolRegistry creates an empty ToolRegistry with the given module name.
func NewToolRegistry(name string) *ToolRegistry {
	return &ToolRegistry{name: name}
}

// Name implements modular.Module.
func (r *ToolRegistry) Name() string { return r.name }

// Init implements modular.Module (no-op; no application wiring required here).
func (r *ToolRegistry) Init(_ interface{}) error { return nil }

// Add appends a tool and its handler to the registry. Safe for concurrent use.
func (r *ToolRegistry) Add(tool *mcpsdk.Tool, h mcpsdk.ToolHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools = append(r.tools, RegisteredTool{Tool: tool, Handler: h})
}

// All returns a snapshot copy of all registered tools in insertion order.
// Callers may freely mutate the returned slice without affecting the registry.
func (r *ToolRegistry) All() []RegisteredTool {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]RegisteredTool, len(r.tools))
	copy(out, r.tools)
	return out
}
