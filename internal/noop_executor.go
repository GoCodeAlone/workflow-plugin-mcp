package internal

import (
	"context"
	"fmt"
)

// noopPipelineExecutor is a stub interfaces.PipelineExecutor that always
// returns an error. It is pre-registered in the shared modular.Application so
// that mcp.ToolTrigger.Configure can resolve the executor without crashing.
//
// In gRPC external plugin mode there is no host engine available to execute
// pipelines — this is the documented v0.1.0 limitation. Any actual tool call
// reaching this executor in gRPC mode will receive a descriptive error.
type noopPipelineExecutor struct{}

func (noopPipelineExecutor) ExecutePipeline(_ context.Context, name string, _ map[string]any) (map[string]any, error) {
	return nil, fmt.Errorf("workflow-plugin-mcp: gRPC mode does not support pipeline execution (pipeline %q); use in-process library mode for full wiring", name)
}
