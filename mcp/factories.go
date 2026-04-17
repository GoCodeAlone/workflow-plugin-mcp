package mcp

import (
	"fmt"

	"github.com/GoCodeAlone/modular"
)

// defaultRegistryModuleName is the conventional service name for the shared
// ToolRegistry when none is explicitly specified in config.
const defaultRegistryModuleName = "mcp.tool-registry"

// ParseServerConfig extracts a ServerConfig from a raw config map.
// Exported so the gRPC adapter in Task 2.5b can reuse it.
//
// Returns an error when required keys are missing or malformed (e.g.
// implementation block is not a map). Callers relying on validation —
// like the gRPC adapter — should surface the error; factories that are
// expected to log-and-retry can ignore it.
func ParseServerConfig(cfg map[string]any) (ServerConfig, error) {
	var sc ServerConfig

	if v, ok := cfg["implementation"]; ok && v != nil {
		impl, ok := v.(map[string]any)
		if !ok {
			return sc, fmt.Errorf("mcp.server config: 'implementation' must be a map, got %T", v)
		}
		sc.Implementation.Name, _ = impl["name"].(string)
		sc.Implementation.Version, _ = impl["version"].(string)
	}

	if v, ok := cfg["registry"]; ok && v != nil {
		reg, ok := v.(string)
		if !ok {
			return sc, fmt.Errorf("mcp.server config: 'registry' must be a string, got %T", v)
		}
		sc.RegistryModuleName = reg
	}
	if sc.RegistryModuleName == "" {
		sc.RegistryModuleName = defaultRegistryModuleName
	}
	return sc, nil
}

// ParseStdioTransportConfig extracts a stdio transport server name from cfg.
// Exported for Task 2.5b reuse. Returns an error if 'server' is present but
// not a string.
func ParseStdioTransportConfig(cfg map[string]any) (string, error) {
	v, ok := cfg["server"]
	if !ok || v == nil {
		return "", nil
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("mcp.stdio_transport config: 'server' must be a string, got %T", v)
	}
	return s, nil
}

// ParseHTTPTransportConfig extracts an HTTPTransportConfig from cfg along with
// the server module name. Exported for Task 2.5b reuse.
func ParseHTTPTransportConfig(cfg map[string]any) (HTTPTransportConfig, string, error) {
	var tc HTTPTransportConfig

	if v, ok := cfg["address"]; ok && v != nil {
		addr, ok := v.(string)
		if !ok {
			return tc, "", fmt.Errorf("mcp.http_transport config: 'address' must be a string, got %T", v)
		}
		tc.Address = addr
	}
	if tc.Address == "" {
		tc.Address = ":8080"
	}

	var serverName string
	if v, ok := cfg["server"]; ok && v != nil {
		s, ok := v.(string)
		if !ok {
			return tc, "", fmt.Errorf("mcp.http_transport config: 'server' must be a string, got %T", v)
		}
		serverName = s
	}
	return tc, serverName, nil
}

func serverModuleFactory(name string, cfg map[string]any) modular.Module {
	sc, _ := ParseServerConfig(cfg)
	return NewServerModule(name, sc)
}

func stdioTransportFactory(name string, cfg map[string]any) modular.Module {
	serverName, _ := ParseStdioTransportConfig(cfg)
	return NewStdioTransportModuleByName(name, serverName)
}

func httpTransportFactory(name string, cfg map[string]any) modular.Module {
	tc, serverName, _ := ParseHTTPTransportConfig(cfg)
	return NewHTTPTransportModuleByName(name, tc, serverName)
}

func toolRegistryFactory(name string, _ map[string]any) modular.Module {
	return NewToolRegistry(name)
}
