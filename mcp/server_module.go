package mcp

import (
	"context"
	"errors"
	"fmt"

	"github.com/GoCodeAlone/modular"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Implementation holds identification metadata for an MCP server.
type Implementation struct {
	Name    string
	Version string
}

// ServerConfig is the configuration for a ServerModule.
type ServerConfig struct {
	Implementation     Implementation
	RegistryModuleName string // defaults to "mcp.tool-registry" if empty
}

// ServerModule is a modular.Module that owns and initialises an MCP server.
type ServerModule struct {
	name   string
	cfg    ServerConfig
	app    modular.Application
	server *mcpsdk.Server
}

var _ modular.Module = (*ServerModule)(nil)
var _ modular.Startable = (*ServerModule)(nil)

// NewServerModule constructs a ServerModule with the given logical name and config.
func NewServerModule(name string, cfg ServerConfig) *ServerModule {
	return &ServerModule{name: name, cfg: cfg}
}

// Name returns the module's unique identifier within the application.
func (m *ServerModule) Name() string { return m.name }

// Init implements modular.Module. It creates the underlying MCP server and
// stashes the application for use in Start.
func (m *ServerModule) Init(app modular.Application) error {
	if m.cfg.Implementation.Name == "" {
		return errors.New("mcp: ServerConfig.Implementation.Name must not be empty")
	}
	m.app = app
	m.server = mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    m.cfg.Implementation.Name,
		Version: m.cfg.Implementation.Version,
	}, nil)
	return nil
}

// registryName returns the effective registry module name, applying the
// default when none was configured.
func (m *ServerModule) registryName() string {
	if m.cfg.RegistryModuleName != "" {
		return m.cfg.RegistryModuleName
	}
	return defaultRegistryModuleName
}

// Start implements modular.Startable. It resolves the ToolRegistry from the
// application service registry and replays all registered tools onto the
// underlying MCP server. Transports must start after ServerModule (declared
// via Dependencies()) so the server is fully equipped before serving begins.
func (m *ServerModule) Start(_ context.Context) error {
	if m.app == nil {
		return fmt.Errorf("mcp: server %q: Start called before Init", m.name)
	}
	regName := m.registryName()
	svc, ok := m.app.SvcRegistry()[regName]
	if !ok {
		return fmt.Errorf("mcp: server %q: ToolRegistry %q not found in service registry", m.name, regName)
	}
	reg, ok := svc.(*ToolRegistry)
	if !ok {
		return fmt.Errorf("mcp: server %q: service %q is not a *ToolRegistry (got %T)", m.name, regName, svc)
	}
	for _, rt := range reg.All() {
		m.server.AddTool(rt.Tool, rt.Handler)
	}
	return nil
}

// Server returns the underlying *mcpsdk.Server, which is non-nil after a
// successful call to Init.
func (m *ServerModule) Server() *mcpsdk.Server { return m.server }

// ToolNames returns the names of all tools currently registered on the
// underlying MCP server. It proxies through the ToolRegistry rather than
// the SDK internals to remain SDK-version agnostic.
// Returns nil if Init has not yet been called.
func (m *ServerModule) ToolNames() []string {
	if m.app == nil {
		return nil
	}
	svc, ok := m.app.SvcRegistry()[m.registryName()]
	if !ok {
		return nil
	}
	reg, ok := svc.(*ToolRegistry)
	if !ok {
		return nil
	}
	tools := reg.All()
	names := make([]string, len(tools))
	for i, rt := range tools {
		names[i] = rt.Tool.Name
	}
	return names
}
