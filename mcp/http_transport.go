package mcp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/GoCodeAlone/modular"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// HTTPTransportConfig holds configuration for an HTTPTransportModule.
type HTTPTransportConfig struct {
	// Address is the TCP address on which the HTTP MCP server listens,
	// e.g. "0.0.0.0:8080" or "127.0.0.1:0".
	Address string
}

// HTTPTransportModule attaches a Streamable-HTTP transport to a ServerModule
// and implements modular.Module, modular.Startable, and modular.Stoppable.
type HTTPTransportModule struct {
	name       string
	cfg        HTTPTransportConfig
	serverName string // used when server is resolved via app at Init time
	server     *ServerModule
	httpSrv    *http.Server
	boundAddr  string
}

// Compile-time interface assertions.
var _ modular.Module = (*HTTPTransportModule)(nil)
var _ modular.Startable = (*HTTPTransportModule)(nil)
var _ modular.Stoppable = (*HTTPTransportModule)(nil)

// NewHTTPTransportModule constructs an HTTPTransportModule with a direct
// ServerModule reference. server must be non-nil and its Init must have been
// called before this module's Init is invoked.
func NewHTTPTransportModule(name string, cfg HTTPTransportConfig, server *ServerModule) *HTTPTransportModule {
	return &HTTPTransportModule{name: name, cfg: cfg, server: server}
}

// NewHTTPTransportModuleByName constructs an HTTPTransportModule that
// resolves its ServerModule from the application registry during Init.
// This is the factory-friendly constructor used by MCPPlugin.
func NewHTTPTransportModuleByName(name string, cfg HTTPTransportConfig, serverName string) *HTTPTransportModule {
	return &HTTPTransportModule{name: name, cfg: cfg, serverName: serverName}
}

// Name implements modular.Module.
func (m *HTTPTransportModule) Name() string { return m.name }

// Address returns the address the HTTP server is actually bound to after Start.
// Before Start it returns an empty string.  Useful when Address is "host:0"
// and the OS assigns an ephemeral port.
func (m *HTTPTransportModule) Address() string { return m.boundAddr }

// Init implements modular.Module.  If constructed via NewHTTPTransportModuleByName,
// it resolves the ServerModule from the application registry; otherwise it
// validates the directly wired ServerModule has been initialised.
func (m *HTTPTransportModule) Init(app modular.Application) error {
	if m.server == nil && m.serverName != "" && app != nil {
		mod := app.GetModule(m.serverName)
		if mod == nil {
			return fmt.Errorf("mcp: http_transport %q: server module %q not found", m.name, m.serverName)
		}
		srv, ok := mod.(*ServerModule)
		if !ok {
			return fmt.Errorf("mcp: http_transport %q: module %q is not a *ServerModule", m.name, m.serverName)
		}
		m.server = srv
	}
	if m.server == nil || m.server.Server() == nil {
		return fmt.Errorf("mcp: http_transport %q: server not wired or not initialised", m.name)
	}
	if m.cfg.Address == "" {
		return fmt.Errorf("mcp: http_transport %q: address is required", m.name)
	}
	return nil
}

// Dependencies implements modular.DependencyAware.  When constructed via
// NewHTTPTransportModuleByName the transport declares the server as a
// dependency so the modular framework initialises and starts it first.
func (m *HTTPTransportModule) Dependencies() []string {
	if m.serverName != "" {
		return []string{m.serverName}
	}
	return nil
}

// Start implements modular.Startable.  It binds the listener synchronously so
// that bind failures (port in use, permission denied) are returned immediately.
// The HTTP server is then served in a background goroutine.
// http.ErrServerClosed is swallowed because it is the expected termination
// signal from Stop.
func (m *HTTPTransportModule) Start(_ context.Context) error {
	handler := mcpsdk.NewStreamableHTTPHandler(func(r *http.Request) *mcpsdk.Server {
		return m.server.Server()
	}, nil)

	ln, err := net.Listen("tcp", m.cfg.Address)
	if err != nil {
		return fmt.Errorf("http_transport %q: listen on %s: %w", m.name, m.cfg.Address, err)
	}
	m.boundAddr = ln.Addr().String()
	m.httpSrv = &http.Server{Handler: handler}

	go func() {
		if err := m.httpSrv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			// Serve errors after bind are logged but can't be returned; they
			// most often mean the listener was closed out from under us.
		}
	}()

	return nil
}

// Stop implements modular.Stoppable.  It performs a graceful HTTP shutdown,
// waiting at most until ctx is cancelled.
func (m *HTTPTransportModule) Stop(ctx context.Context) error {
	if m.httpSrv == nil {
		return nil
	}
	return m.httpSrv.Shutdown(ctx)
}
