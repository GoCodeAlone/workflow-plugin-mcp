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
// Start spawns a goroutine that runs the MCP server over stdin/stdout.
// Stop cancels the internal context that drives the Run goroutine, so the
// transport shuts down regardless of whether the caller cancels the start
// context.
type StdioTransportModule struct {
	name   string
	server *ServerModule
	cancel context.CancelFunc
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
// MCP server over a StdioTransport.  The goroutine exits when the internal
// cancel context is cancelled — either by Stop or by the parent ctx.
// Start returns an error if called while already running.
func (m *StdioTransportModule) Start(ctx context.Context) error {
	if m.cancel != nil {
		return fmt.Errorf("mcp: stdio_transport %q: already started", m.name)
	}
	runCtx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	go func() {
		_ = m.server.Server().Run(runCtx, &mcpsdk.StdioTransport{})
	}()
	return nil
}

// Stop implements modular.Stoppable.  It cancels the context that drives the
// Run goroutine, triggering an orderly shutdown.
func (m *StdioTransportModule) Stop(_ context.Context) error {
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	return nil
}
