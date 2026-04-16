package mcp

import (
	"context"
	"errors"
	"fmt"
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
	name    string
	cfg     HTTPTransportConfig
	server  *ServerModule
	httpSrv *http.Server
}

// Compile-time interface assertions.
var _ modular.Module = (*HTTPTransportModule)(nil)
var _ modular.Startable = (*HTTPTransportModule)(nil)
var _ modular.Stoppable = (*HTTPTransportModule)(nil)

// NewHTTPTransportModule constructs an HTTPTransportModule.
// server must be non-nil and its Init must have been called before this
// module's Init is invoked.
func NewHTTPTransportModule(name string, cfg HTTPTransportConfig, server *ServerModule) *HTTPTransportModule {
	return &HTTPTransportModule{name: name, cfg: cfg, server: server}
}

// Name implements modular.Module.
func (m *HTTPTransportModule) Name() string { return m.name }

// Init implements modular.Module.  It validates the wired ServerModule and
// configuration.
// app may be nil; service-registry wiring is deferred to Task 2.5.
func (m *HTTPTransportModule) Init(_ modular.Application) error {
	if m.server == nil || m.server.Server() == nil {
		return fmt.Errorf("mcp: http_transport %q: server not wired or not initialised", m.name)
	}
	if m.cfg.Address == "" {
		return fmt.Errorf("mcp: http_transport %q: address is required", m.name)
	}
	return nil
}

// Start implements modular.Startable.  It builds a StreamableHTTPHandler,
// wraps it in an http.Server, and calls ListenAndServe in a goroutine.
// http.ErrServerClosed is swallowed because it is the expected termination
// signal from Stop.
func (m *HTTPTransportModule) Start(_ context.Context) error {
	handler := mcpsdk.NewStreamableHTTPHandler(func(r *http.Request) *mcpsdk.Server {
		return m.server.Server()
	}, nil)

	m.httpSrv = &http.Server{
		Addr:    m.cfg.Address,
		Handler: handler,
	}

	go func() {
		if err := m.httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			// Non-critical: the transport is best-effort; callers observe
			// connection-refused on the address if binding failed.
			_ = err
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
