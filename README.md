# workflow-plugin-mcp

Model Context Protocol (MCP) plugin for the [GoCodeAlone/workflow](https://github.com/GoCodeAlone/workflow) engine. Exposes workflow pipelines as MCP tools that an LLM can call, via stdio or streamable HTTP transport.

Two consumption modes are supported:

1. **In-process library** — import `github.com/GoCodeAlone/workflow-plugin-mcp/mcp` and register `mcp.New()` with a `StdEngine`. This is the recommended mode and the one used by the [zoom-mcp](https://github.com/GoCodeAlone/zoom-mcp) service.
2. **External gRPC plugin** — a standalone binary loaded by the engine via `plugin.json`. Scoped for simple topologies only; cross-module wiring is not available across the subprocess boundary in v0.1.0 (see [Limitations](#v010-limitations)).

## Provided module and trigger types

| Type | Kind | Purpose |
|---|---|---|
| `mcp.server` | module | Owns the MCP server instance. Replays registered tools onto the server at `Start`. |
| `mcp.tool_registry` | module | Shared in-memory registry of tools pending registration with the server. |
| `mcp.stdio_transport` | module | Serves the MCP server on stdio. Useful for LLM clients that launch the server as a subprocess. |
| `mcp.http_transport` | module | Serves the MCP server over streamable HTTP. |
| `mcp.tool` | trigger | Registers a workflow pipeline as a named MCP tool with a JSON-schema-validated input. |

## YAML shape

```yaml
modules:
  - name: mcp.tool-registry
    type: mcp.tool_registry

  - name: my-server
    type: mcp.server
    config:
      implementation:
        name: my-service
        version: "1.0.0"
      registry: mcp.tool-registry        # optional; defaults to "mcp.tool-registry"

  - name: my-stdio
    type: mcp.stdio_transport
    config:
      server: my-server

  # or:
  - name: my-http
    type: mcp.http_transport
    config:
      server: my-server
      address: "127.0.0.1:8080"          # optional; defaults to ":8080"

pipelines:
  greet:
    trigger:
      type: mcp.tool
      config:
        server: my-server
        registry: mcp.tool-registry
        name: greet
        description: "Greet a person by name"
        input_schema:
          type: object
          properties:
            name: { type: string }
          required: [name]
        # or reference an external schema file:
        # input_schema:
        #   $ref: "./schemas/greet.input.json"
        config_dir: "./config"           # required when $ref is relative; see below
    steps: [ ... ]
```

### Schema handling

`input_schema` may be provided inline (as above) or via a single-key `{"$ref": "./path/to/schema.json"}` that points to an external JSON Schema file.

The `config_dir` key on the trigger config is the base directory used to resolve relative `$ref` paths. The workflow engine injects a `_config_dir` key for module configs today but **not** for trigger configs — callers must pass `config_dir` explicitly on each `mcp.tool` trigger that uses `$ref`. Absolute `$ref` paths and inline schemas do not require it.

The top-level schema type must be `"object"` — the MCP SDK rejects anything else.

## Usage — in-process library mode

```go
package main

import (
    "context"
    "log/slog"

    "github.com/GoCodeAlone/modular"
    "github.com/GoCodeAlone/workflow"
    "github.com/GoCodeAlone/workflow/config"

    "github.com/GoCodeAlone/workflow-plugin-mcp/mcp"
)

func run(ctx context.Context, yamlCfg string) error {
    cfg, err := config.LoadFromString(yamlCfg)
    if err != nil { return err }

    engine, err := workflow.NewEngineBuilder().
        WithLogger(loggerAdapter(slog.Default())).
        WithAllDefaults().
        WithPlugin(mcp.New()).
        BuildFromConfig(cfg)
    if err != nil { return err }

    return engine.Start(ctx)
}

func loggerAdapter(l *slog.Logger) modular.Logger { /* ... */ }
```

`mcp.New()` returns a `*mcp.MCPPlugin` that registers all module and trigger factories. During engine startup:

1. Each `mcp.tool` trigger `Configure`'s itself, compiling its input schema and appending its `(Tool, Handler)` pair to the shared `mcp.tool_registry`.
2. `mcp.server` modules `Start` after all registry writes; each replays the registry onto its underlying `mcpsdk.Server` via `server.AddTool(...)`.
3. Transports `Start` after their server dependency (declared via `Dependencies()`), so clients never see a partially-populated tool list.

## Usage — external gRPC plugin mode

The `cmd/workflow-plugin-mcp/main.go` binary calls `sdk.Serve(internal.NewPlugin())`, exposing the same four module types and trigger type over the workflow SDK's gRPC protocol. `plugin.json` declares the capabilities for the workflow registry to discover.

```sh
go build -o workflow-plugin-mcp ./cmd/workflow-plugin-mcp
```

Drop the binary alongside `plugin.json` into the host workflow's plugin directory. The engine will spawn it as a subprocess when a pipeline references any of its types.

### v0.1.0 limitations

The external gRPC subprocess has no access to the host's modular service registry. This means:

- **Cross-module wiring does not work across the gRPC boundary.** A transport running in the subprocess cannot resolve a server module running in a different process. All modules in a gRPC plugin instance share one in-subprocess `modular.Application`, so a "one server + one transport + N tools" topology inside a single plugin instance works, but multi-plugin wiring does not.
- **Pipeline dispatch is not available in gRPC mode.** The subprocess has no host `PipelineExecutor`; the plugin pre-seeds a no-op executor so `Configure` returns cleanly, and tool calls surface a `"pipeline execution not available in gRPC mode"` error at call time.
- **Full functionality requires the in-process library mode.** Use `mcp.New()` from your host binary. The gRPC mode is provided for discoverability against the workflow plugin registry; production use should prefer in-process.

These limitations will be addressed in a future release by wiring `sdk.TriggerCallback` to a bridge executor that marshals pipeline dispatch back to the host.

## Testing with `InMemoryTransport`

The MCP SDK ships `mcp.NewInMemoryTransports()` which returns a paired `(client, server)` transport — ideal for integration tests that exercise the full engine → server → tool-call path without binding a real socket. See `mcp/e2e_test.go` for the pattern:

```go
clientTr, serverTr := mcpsdk.NewInMemoryTransports()
go serverMod.Server().Run(ctx, serverTr)

client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "t", Version: "0"}, nil)
session, _ := client.Connect(ctx, clientTr, nil)

result, _ := session.CallTool(ctx, &mcpsdk.CallToolParams{
    Name:      "greet",
    Arguments: map[string]any{"name": "world"},
})
```

## Development

```sh
go build ./...
go test ./... -race -count=1
```

Cross-compile the gRPC binary for a Linux deployment:

```sh
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" \
    -o workflow-plugin-mcp ./cmd/workflow-plugin-mcp
```

## License

MIT — see [LICENSE](LICENSE).
