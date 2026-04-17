package mcp

import (
	"sync"

	"github.com/GoCodeAlone/modular"
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

var _ modular.Module = (*ToolRegistry)(nil)
var _ modular.ServiceAware = (*ToolRegistry)(nil)

// NewToolRegistry creates an empty ToolRegistry with the given module name.
func NewToolRegistry(name string) *ToolRegistry {
	return &ToolRegistry{name: name}
}

// Name implements modular.Module.
func (r *ToolRegistry) Name() string { return r.name }

// Init implements modular.Module (no-op; service registration is via ProvidesServices).
func (r *ToolRegistry) Init(_ modular.Application) error { return nil }

// ProvidesServices implements modular.ServiceAware. It registers the registry
// itself under its module name so ServerModule and ToolTrigger can look it up.
func (r *ToolRegistry) ProvidesServices() []modular.ServiceProvider {
	return []modular.ServiceProvider{
		{Name: r.name, Instance: r},
	}
}

// RequiresServices implements modular.ServiceAware (no requirements).
func (r *ToolRegistry) RequiresServices() []modular.ServiceDependency { return nil }

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
