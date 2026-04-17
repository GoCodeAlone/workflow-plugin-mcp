package internal

import (
	"context"

	"github.com/GoCodeAlone/modular"
	"github.com/GoCodeAlone/workflow-plugin-mcp/mcp"
	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

// triggerAdapter wraps a *mcp.ToolTrigger as an sdk.TriggerInstance.
//
// v0.1.0 limitation: The sdk.TriggerInstance interface only has Start/Stop;
// there is no Configure step after construction. Therefore Configure is called
// eagerly at construction time (inside CreateTrigger). If Configure fails, the
// error is surfaced immediately so the host sees it before attempting Start.
//
// The sdk.TriggerCallback cb is the SDK's intended bridge for a gRPC trigger
// to dispatch actions on the host engine. In v0.1.0 it is stored but not wired:
// ToolTrigger.Configure captures the pre-seeded noopPipelineExecutor via its
// handler closure, so tool calls surface a "not available in gRPC mode" error
// at call time. A future v0.2.0 change should build a cb-backed executor shim
// that marshals the pipeline name + args back to the host via cb, replacing
// the noop executor before Configure runs.
type triggerAdapter struct {
	trigger *mcp.ToolTrigger
	cb      sdk.TriggerCallback // v0.2.0: wire to a cb-backed executor (see type doc)
}

// newTriggerAdapter configures t eagerly and returns an sdk.TriggerInstance.
// Returns an error if Configure fails (e.g. missing required "name" key or
// malformed "workflowType").
func newTriggerAdapter(app modular.Application, t *mcp.ToolTrigger, cfg map[string]any, cb sdk.TriggerCallback) (sdk.TriggerInstance, error) {
	if err := t.Configure(app, cfg); err != nil {
		return nil, err
	}
	return &triggerAdapter{trigger: t, cb: cb}, nil
}

// Start implements sdk.TriggerInstance.
func (a *triggerAdapter) Start(ctx context.Context) error {
	return a.trigger.Start(ctx)
}

// Stop implements sdk.TriggerInstance.
func (a *triggerAdapter) Stop(ctx context.Context) error {
	return a.trigger.Stop(ctx)
}
