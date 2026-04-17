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
// The sdk.TriggerCallback is stored for future use but is not wired in v0.1.0;
// the in-process pipeline executor handles dispatch.
type triggerAdapter struct {
	trigger *mcp.ToolTrigger
	cb      sdk.TriggerCallback // unused in v0.1.0, stored for future wiring
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
