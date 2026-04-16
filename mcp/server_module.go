package mcp

import (
	"errors"

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
	Implementation Implementation
}

// ServerModule is a modular.Module that owns and initialises an MCP server.
type ServerModule struct {
	name   string
	cfg    ServerConfig
	server *mcpsdk.Server
}

var _ modular.Module = (*ServerModule)(nil)

// NewServerModule constructs a ServerModule with the given logical name and config.
func NewServerModule(name string, cfg ServerConfig) *ServerModule {
	return &ServerModule{name: name, cfg: cfg}
}

// Name returns the module's unique identifier within the application.
func (m *ServerModule) Name() string { return m.name }

// Init implements modular.Module. It creates the underlying MCP server.
// app may be nil; service-registry wiring is deferred to Task 2.5.
func (m *ServerModule) Init(_ modular.Application) error {
	if m.cfg.Implementation.Name == "" {
		return errors.New("mcp: ServerConfig.Implementation.Name must not be empty")
	}
	m.server = mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    m.cfg.Implementation.Name,
		Version: m.cfg.Implementation.Version,
	}, nil)
	return nil
}

// Server returns the underlying *mcpsdk.Server, which is non-nil after a
// successful call to Init.
func (m *ServerModule) Server() *mcpsdk.Server { return m.server }
