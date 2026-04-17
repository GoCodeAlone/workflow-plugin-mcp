package internal

import (
	"context"

	"github.com/GoCodeAlone/modular"
	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

// moduleAdapter wraps a modular.Module as an sdk.ModuleInstance, bridging the
// gRPC external plugin lifecycle (Init/Start/Stop with no-arg Init) to the
// modular framework lifecycle (Init takes a modular.Application).
//
// A shared modular.Application is injected at construction time so that the
// underlying module can store it for later use in Start/Stop — even though the
// gRPC subprocess does not have access to the host's service registry.
type moduleAdapter struct {
	app modular.Application
	mod modular.Module
}

// newModuleAdapter creates an sdk.ModuleInstance wrapping the given module.
// The provided app is passed through to mod.Init when Init() is called.
func newModuleAdapter(app modular.Application, mod modular.Module) sdk.ModuleInstance {
	return &moduleAdapter{app: app, mod: mod}
}

// Init implements sdk.ModuleInstance. Calls mod.Init with the shared app.
func (a *moduleAdapter) Init() error {
	return a.mod.Init(a.app)
}

// Start implements sdk.ModuleInstance. Calls mod.Start if it implements
// modular.Startable; otherwise is a no-op.
func (a *moduleAdapter) Start(ctx context.Context) error {
	if s, ok := a.mod.(modular.Startable); ok {
		return s.Start(ctx)
	}
	return nil
}

// Stop implements sdk.ModuleInstance. Calls mod.Stop if it implements
// modular.Stoppable; otherwise is a no-op.
func (a *moduleAdapter) Stop(ctx context.Context) error {
	if s, ok := a.mod.(modular.Stoppable); ok {
		return s.Stop(ctx)
	}
	return nil
}
