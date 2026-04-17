package mcp

import "github.com/GoCodeAlone/modular"

// ParseServerConfig extracts a ServerConfig from a raw config map.
// Exported so the gRPC adapter in Task 2.5b can reuse it.
func ParseServerConfig(cfg map[string]any) ServerConfig {
	var sc ServerConfig
	if impl, ok := cfg["implementation"].(map[string]any); ok {
		sc.Implementation.Name, _ = impl["name"].(string)
		sc.Implementation.Version, _ = impl["version"].(string)
	}
	sc.RegistryModuleName, _ = cfg["registry"].(string)
	if sc.RegistryModuleName == "" {
		sc.RegistryModuleName = "mcp.tool-registry"
	}
	return sc
}

// ParseStdioTransportConfig extracts a stdio transport server name from cfg.
// Exported for Task 2.5b reuse.
func ParseStdioTransportConfig(cfg map[string]any) string {
	s, _ := cfg["server"].(string)
	return s
}

// ParseHTTPTransportConfig extracts an HTTPTransportConfig from cfg.
// Exported for Task 2.5b reuse.
func ParseHTTPTransportConfig(cfg map[string]any) HTTPTransportConfig {
	addr, _ := cfg["address"].(string)
	if addr == "" {
		addr = ":8080"
	}
	return HTTPTransportConfig{Address: addr}
}

func serverModuleFactory(name string, cfg map[string]any) modular.Module {
	return NewServerModule(name, ParseServerConfig(cfg))
}

func stdioTransportFactory(name string, cfg map[string]any) modular.Module {
	serverName := ParseStdioTransportConfig(cfg)
	return NewStdioTransportModuleByName(name, serverName)
}

func httpTransportFactory(name string, cfg map[string]any) modular.Module {
	serverName, _ := cfg["server"].(string)
	return NewHTTPTransportModuleByName(name, ParseHTTPTransportConfig(cfg), serverName)
}

func toolRegistryFactory(name string, _ map[string]any) modular.Module {
	return NewToolRegistry(name)
}
