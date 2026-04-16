package mcp

import (
	"context"
	"fmt"

	"github.com/GoCodeAlone/modular"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// StdioTransportModule attaches a stdio transport to a ServerModule and
// implements modular.Module, modular.Startable, and modular.Stoppable.
//
// Start spawns a goroutine that runs the MCP server over stdin/stdout.  The
// goroutine unwinds automatically when the context passed to Start is
// cancelled, so Stop is a deliberate no-op — the lifecycle is driven
// entirely by context cancellation (matching the pattern used by the
// Workflow http server module).
type StdioTransportModule struct {
	name   string
	server *ServerModule
}

// Compile-time interface assertions.
var _ modular.Module = (*StdioTransportModule)(nil)
var _ modular.Startable = (*StdioTransportModule)(nil)
var _ modular.Stoppable = (*StdioTransportModule)(nil)

// NewStdioTransportModule constructs a StdioTransportModule.
// server must be non-nil and its Init must have been called before this
// module's Init is invoked.
func NewStdioTransportModule(name string, server *ServerModule) *StdioTransportModule {
	return &StdioTransportModule{name: name, server: server}
}

// Name implements modular.Module.
func (m *StdioTransportModule) Name() string { return m.name }

// Init implements modular.Module.  It validates that the wired ServerModule
// has been initialised (i.e. server.Server() is non-nil).
// app may be nil; service-registry wiring is deferred to Task 2.5.
func (m *StdioTransportModule) Init(_ modular.Application) error {
	if m.server == nil || m.server.Server() == nil {
		return fmt.Errorf("mcp: stdio_transport %q: server not wired or not initialised", m.name)
	}
	return nil
}

// Start implements modular.Startable.  It spawns a goroutine that runs the
// MCP server over a StdioTransport.  The goroutine exits when ctx is
// cancelled.  Start itself returns nil immediately.
func (m *StdioTransportModule) Start(ctx context.Context) error {
	go func() {
		_ = m.server.Server().Run(ctx, &mcpsdk.StdioTransport{})
	}()
	return nil
}

// Stop implements modular.Stoppable.  Because the Run goroutine unwinds on
// context cancellation, there is nothing additional to do here.
func (m *StdioTransportModule) Stop(_ context.Context) error {
	return nil
}
