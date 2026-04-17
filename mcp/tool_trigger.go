package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoCodeAlone/modular"
	"github.com/GoCodeAlone/workflow/interfaces"
	jsonschema "github.com/santhosh-tekuri/jsonschema/v5"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// registeredPipelineTool holds the per-pipeline metadata stashed by Configure.
type registeredPipelineTool struct {
	pipelineName string
	schema       *jsonschema.Schema // compiled input schema; may be nil if none provided
}

// ToolTriggerConfig is the structured form of a trigger configuration block
// for the mcp.tool trigger type. In YAML-driven workflows the engine deserialises
// trigger config into a map[string]any; ToolTrigger.Configure reads the keys
// directly from that map.
//
// Field names here document the supported YAML keys for reference only —
// no struct-tag based decode is performed; see Configure for the actual parsing.
type ToolTriggerConfig struct {
	// ServerModuleName is the modular service name of the ServerModule
	// that owns the MCP server to register tools on ("server" key in YAML).
	ServerModuleName string

	// RegistryModuleName is the modular service name of the ToolRegistry.
	// Defaults to "mcp.tool-registry" if omitted.
	RegistryModuleName string

	// ToolName is the MCP tool name exposed to LLM clients ("name" key).
	ToolName string

	// Description is the human-readable tool description.
	Description string

	// InputSchema is an inline JSON-schema map or a single-key
	// {$ref: "./path/to/schema.json"} map. Top-level type must be "object".
	InputSchema map[string]any

	// OutputSchema is an optional inline JSON-schema for the tool output.
	// Currently recorded but not validated at call time.
	OutputSchema map[string]any
}

// ToolTrigger is an interfaces.Trigger that registers a pipeline as an MCP
// tool. Each call to Configure (one per pipeline that names this trigger)
// registers one new tool on the shared ToolRegistry. The ServerModule then
// picks those tools up during its Start phase (Task 2.5).
//
// NOTE: _config_dir is NOT injected into pipeline trigger configs by the engine
// today (only module configs receive it). As a result, $ref paths in
// input_schema must be absolute, or the caller must supply a "config_dir" key
// in the trigger config map. This limitation will be addressed in a later
// upstream change.
type ToolTrigger struct {
	// name is the modular module name for this trigger instance.
	name string

	// serverModuleName is captured from the last Configure call for Task 2.5
	// wiring, where the ServerModule replays ToolRegistry entries onto itself
	// during Start. Not used today.
	serverModuleName string

	// tools is a map from tool name to the pipeline/schema metadata stashed on
	// Configure. Primarily useful for observability and testing; also used to
	// reject duplicate tool-name registration.
	tools map[string]*registeredPipelineTool

	// executor is resolved from the service registry on first Configure call.
	executor interfaces.PipelineExecutor

	// registry is resolved from the service registry on first Configure call.
	registry *ToolRegistry
}

// Compile-time assertion: *ToolTrigger must satisfy interfaces.Trigger.
var _ interfaces.Trigger = (*ToolTrigger)(nil)

// NewToolTrigger creates an uninitialised ToolTrigger. The returned instance
// satisfies modular.Module and interfaces.Trigger; its service dependencies are
// resolved during Configure.
func NewToolTrigger() *ToolTrigger {
	return &ToolTrigger{
		name:  "mcp.tool",
		tools: make(map[string]*registeredPipelineTool),
	}
}

// Name implements modular.Module.
func (t *ToolTrigger) Name() string { return t.name }

// Dependencies implements modular.Module. The trigger has no declared module
// dependencies; it resolves its runtime dependencies (ToolRegistry,
// PipelineExecutor) from the service registry inside Configure.
func (t *ToolTrigger) Dependencies() []string { return nil }

// Init implements modular.Module. No-op; wiring is deferred to Configure.
func (t *ToolTrigger) Init(_ modular.Application) error { return nil }

// Start implements modular.Startable. No-op; the tool is already registered
// with the ToolRegistry during Configure.
func (t *ToolTrigger) Start(_ context.Context) error { return nil }

// Stop implements modular.Stoppable. No-op.
func (t *ToolTrigger) Stop(_ context.Context) error { return nil }

// Configure implements interfaces.Trigger.
//
// It is called once per pipeline that declares this trigger type. The engine
// injects a "workflowType" key of the form "pipeline:<name>" before calling
// Configure so the trigger knows which pipeline to dispatch to.
//
// Supported keys in triggerConfig (map[string]any):
//
//	workflowType   string  injected by engine: "pipeline:<pipelineName>"
//	server         string  modular service name of the ServerModule
//	registry       string  modular service name of the ToolRegistry (default: "mcp.tool-registry")
//	name           string  MCP tool name
//	description    string  human-readable description
//	input_schema   map     inline JSON-schema or {$ref: "./path.json"}
//	output_schema  map     optional output schema (not validated at call time)
//	config_dir     string  base directory for $ref resolution (see type doc)
func (t *ToolTrigger) Configure(app modular.Application, triggerConfig any) error {
	cfg, ok := triggerConfig.(map[string]any)
	if !ok {
		return fmt.Errorf("mcp.tool trigger: triggerConfig must be map[string]any, got %T", triggerConfig)
	}

	// --- 1. Extract pipeline name from workflowType ---
	pipelineName, err := extractPipelineName(cfg)
	if err != nil {
		return fmt.Errorf("mcp.tool trigger: %w", err)
	}

	// --- 2. Extract scalar config values ---
	serverName := stringVal(cfg, "server")
	registryName := stringVal(cfg, "registry")
	if registryName == "" {
		registryName = "mcp.tool-registry"
	}
	toolName := stringVal(cfg, "name")
	if toolName == "" {
		return fmt.Errorf("mcp.tool trigger: 'name' (tool name) is required")
	}
	if existing, ok := t.tools[toolName]; ok {
		return fmt.Errorf("mcp.tool trigger: tool name %q already registered (for pipeline %q)", toolName, existing.pipelineName)
	}
	description := stringVal(cfg, "description")
	configDir := stringVal(cfg, "config_dir")

	// --- 3. Resolve input schema ---
	inputSchemaRaw, _ := cfg["input_schema"].(map[string]any)
	inputSchema, err := resolveSchema(inputSchemaRaw, configDir)
	if err != nil {
		return fmt.Errorf("mcp.tool trigger %q: input_schema: %w", toolName, err)
	}

	// --- 4. Validate top-level type == "object" ---
	if inputSchema != nil {
		if err := requireObjectSchema(inputSchema); err != nil {
			return fmt.Errorf("mcp.tool trigger %q: %w", toolName, err)
		}
	}

	// --- 5. Compile input schema ---
	var compiledSchema *jsonschema.Schema
	if inputSchema != nil {
		compiledSchema, err = compileSchema(inputSchema)
		if err != nil {
			return fmt.Errorf("mcp.tool trigger %q: compile input_schema: %w", toolName, err)
		}
	}

	// --- 6. Look up ToolRegistry ---
	registry, err := t.resolveRegistry(app, registryName)
	if err != nil {
		return fmt.Errorf("mcp.tool trigger %q: %w", toolName, err)
	}

	// --- 7. Look up PipelineExecutor ---
	executor, err := t.resolveExecutor(app)
	if err != nil {
		return fmt.Errorf("mcp.tool trigger %q: %w", toolName, err)
	}

	// --- 8. Build and register the handler ---
	handler := t.buildHandler(toolName, pipelineName, compiledSchema, executor)

	tool := &mcpsdk.Tool{
		Name:        toolName,
		Description: description,
		InputSchema: inputSchema,
	}
	registry.Add(tool, handler)

	// --- 9. Stash for observability / testing ---
	t.serverModuleName = serverName
	t.tools[toolName] = &registeredPipelineTool{
		pipelineName: pipelineName,
		schema:       compiledSchema,
	}

	return nil
}

// buildHandler constructs the mcpsdk.ToolHandler closure for the given pipeline.
func (t *ToolTrigger) buildHandler(
	toolName string,
	pipelineName string,
	compiledSchema *jsonschema.Schema,
	executor interfaces.PipelineExecutor,
) mcpsdk.ToolHandler {
	return func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		// 1. Parse arguments.
		var args map[string]any
		if len(req.Params.Arguments) > 0 {
			if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
				return &mcpsdk.CallToolResult{
					Content: []mcpsdk.Content{
						&mcpsdk.TextContent{Text: "invalid input: failed to parse arguments: " + err.Error()},
					},
					IsError: true,
				}, nil
			}
		}
		if args == nil {
			args = make(map[string]any)
		}

		// 2. Validate against compiled schema.
		if compiledSchema != nil {
			if err := compiledSchema.Validate(args); err != nil {
				return &mcpsdk.CallToolResult{
					Content: []mcpsdk.Content{
						&mcpsdk.TextContent{Text: "invalid input: " + err.Error()},
					},
					IsError: true,
				}, nil
			}
		}

		// 3. Dispatch pipeline.
		output, err := executor.ExecutePipeline(ctx, pipelineName, map[string]any{
			"input": args,
			"meta":  map[string]any{"tool_name": toolName},
		})
		if err != nil {
			return &mcpsdk.CallToolResult{
				Content: []mcpsdk.Content{
					&mcpsdk.TextContent{Text: "pipeline error: " + err.Error()},
				},
				IsError: true,
			}, nil
		}

		// 4. Marshal and return success.
		b, err := json.Marshal(output)
		if err != nil {
			return &mcpsdk.CallToolResult{
				Content: []mcpsdk.Content{
					&mcpsdk.TextContent{Text: "failed to marshal pipeline output: " + err.Error()},
				},
				IsError: true,
			}, nil
		}

		return &mcpsdk.CallToolResult{
			Content:           []mcpsdk.Content{&mcpsdk.TextContent{Text: string(b)}},
			StructuredContent: output,
			IsError:           false,
		}, nil
	}
}

// resolveRegistry looks up *ToolRegistry in the service registry.
func (t *ToolTrigger) resolveRegistry(app modular.Application, name string) (*ToolRegistry, error) {
	svc, ok := app.SvcRegistry()[name]
	if !ok {
		return nil, fmt.Errorf("ToolRegistry %q not found in service registry", name)
	}
	reg, ok := svc.(*ToolRegistry)
	if !ok {
		return nil, fmt.Errorf("service %q is not a *ToolRegistry (got %T)", name, svc)
	}
	// Cache on first resolution; all subsequent calls must yield the same instance.
	t.registry = reg
	return reg, nil
}

// resolveExecutor looks up interfaces.PipelineExecutor in the service registry.
// The engine registers exactly one executor per application, so first-found
// iteration is safe and matches the upstream convention in
// workflow/module/trigger_mcp_tool.go.
func (t *ToolTrigger) resolveExecutor(app modular.Application) (interfaces.PipelineExecutor, error) {
	for _, svc := range app.SvcRegistry() {
		if exec, ok := svc.(interfaces.PipelineExecutor); ok {
			t.executor = exec
			return exec, nil
		}
	}
	return nil, fmt.Errorf("no interfaces.PipelineExecutor found in service registry")
}

// --- helpers ---

// extractPipelineName pulls the pipeline name out of the "workflowType"
// key injected by the engine ("pipeline:<name>").
func extractPipelineName(cfg map[string]any) (string, error) {
	wt, _ := cfg["workflowType"].(string)
	if wt == "" {
		return "", fmt.Errorf("'workflowType' key is missing or empty (engine did not inject it)")
	}
	const prefix = "pipeline:"
	if !strings.HasPrefix(wt, prefix) {
		return "", fmt.Errorf("'workflowType' %q has unexpected format; expected 'pipeline:<name>'", wt)
	}
	name := strings.TrimPrefix(wt, prefix)
	if name == "" {
		return "", fmt.Errorf("pipeline name is empty in workflowType %q", wt)
	}
	return name, nil
}

// stringVal extracts a string value from a map by key, returning "" if absent
// or if the value is not a string.
func stringVal(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

// resolveSchema resolves the schema map, following a single $ref if present.
// If schemaMap is nil or empty it returns (nil, nil).
// If configDir is non-empty, relative $ref paths are resolved against it;
// otherwise they are treated as absolute or CWD-relative.
func resolveSchema(schemaMap map[string]any, configDir string) (map[string]any, error) {
	if len(schemaMap) == 0 {
		return nil, nil
	}

	ref, hasRef := schemaMap["$ref"].(string)
	if !hasRef {
		return schemaMap, nil
	}

	// Resolve the path.
	path := ref
	if !filepath.IsAbs(path) {
		if configDir != "" {
			path = filepath.Join(configDir, path)
		}
		// else: treat as relative to CWD; os.ReadFile handles that.
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read $ref %q: %w", ref, err)
	}

	var loaded map[string]any
	if err := json.Unmarshal(data, &loaded); err != nil {
		return nil, fmt.Errorf("parse $ref %q: %w", ref, err)
	}
	return loaded, nil
}

// requireObjectSchema returns an error if the schema map does not have
// "type": "object" at its top level. The MCP SDK requires object-typed input
// schemas.
func requireObjectSchema(schema map[string]any) error {
	t, _ := schema["type"].(string)
	if t != "object" {
		return fmt.Errorf("input_schema top-level 'type' must be 'object', got %q", t)
	}
	return nil
}

// compileSchema compiles a schema map using santhosh-tekuri/jsonschema/v5.
// It marshals the map to JSON, registers it as a resource with a synthetic
// URL, then compiles.
func compileSchema(schema map[string]any) (*jsonschema.Schema, error) {
	b, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("marshal schema: %w", err)
	}
	const syntheticURL = "https://mcp.tool/input-schema"
	c := jsonschema.NewCompiler()
	if err := c.AddResource(syntheticURL, strings.NewReader(string(b))); err != nil {
		return nil, fmt.Errorf("add schema resource: %w", err)
	}
	return c.Compile(syntheticURL)
}
