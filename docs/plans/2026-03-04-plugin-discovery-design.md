# Plugin Discovery and Loading API — Design

## Overview

Enable Flock to load external plugins at startup. External plugins are standalone executables that communicate with Flock over JSON-RPC 2.0 on stdin/stdout. They implement the same capabilities as built-in plugins (runtime routing, service management) but run as child processes.

## Discovery

Flock scans `~/.config/flock/plugins/` (platform-aware via existing `config` package) at startup. Each subdirectory is a plugin containing a manifest and executable:

```
~/.config/flock/plugins/
  flock-node/
    plugin.json
    flock-node          # executable (flock-node.exe on Windows)
  flock-ruby/
    plugin.json
    flock-ruby
```

### Manifest Format (`plugin.json`)

```json
{
  "id": "flock-node",
  "name": "Flock Node",
  "version": "0.1.0",
  "executable": "flock-node",
  "capabilities": ["runtime"],
  "minFlockVersion": "0.1.0"
}
```

- `id` — unique identifier, must not conflict with built-in plugin IDs
- `name` — human-readable display name
- `version` — semver version string
- `executable` — filename of the executable within the plugin directory
- `capabilities` — array of `"runtime"` (handles site requests) and/or `"service"` (manages background services)
- `minFlockVersion` — minimum compatible Flock version

### Validation

- Required fields: `id`, `name`, `version`, `executable`, `capabilities`
- Executable must exist and be executable
- No duplicate IDs with built-in plugins
- Invalid manifests are logged and skipped (non-fatal)

## Loading — ExternalPlugin Adapter

A new `ExternalPlugin` struct implements `Plugin`, `RuntimePlugin`, and `ServicePlugin` by proxying method calls to the subprocess over JSON-RPC 2.0.

The existing `Manager` sees no difference between built-in and external plugins. Registration happens in `core.go` after built-in plugins, giving built-ins priority for upstream resolution.

## JSON-RPC Protocol

Communication is JSON-RPC 2.0 over stdin/stdout. Flock sends requests, the plugin responds. Stderr is captured for logging.

### Methods

| Method | Params | Returns | Used by |
|--------|--------|---------|---------|
| `plugin.init` | `{sites: Site[]}` | `{}` | All plugins |
| `plugin.start` | `{}` | `{}` | All plugins |
| `plugin.stop` | `{}` | `{}` | All plugins |
| `plugin.handles` | `{site: Site}` | `{handles: bool}` | RuntimePlugin |
| `plugin.upstreamFor` | `{site: Site}` | `{upstream: string}` | RuntimePlugin |
| `plugin.serviceStatus` | `{}` | `{status: ServiceStatus}` | ServicePlugin |
| `plugin.startService` | `{}` | `{}` | ServicePlugin |
| `plugin.stopService` | `{}` | `{}` | ServicePlugin |

### Lifecycle

1. Flock spawns the process on `Init()` and sends `plugin.init`
2. `Start()` / `Stop()` map directly to RPC calls
3. On `Stop()`, after the RPC response, Flock sends SIGTERM and waits (with timeout) for the process to exit
4. If the process crashes unexpectedly, the plugin is marked degraded

### Error Handling

- JSON-RPC errors map to Go errors
- Timeouts on any call (10 seconds) mark the plugin degraded
- Same degradation behavior as built-in plugin failures

## Integration with Core

### Discovery Phase

A new `discovery` package:
- `discovery.Scan(pluginsDir) -> []PluginManifest` — reads and validates each `plugin.json`

### Wiring in `core.go`

After registering built-in plugins:

```go
// Built-in plugins (existing)
pluginMgr.Register(sslPlugin)
pluginMgr.Register(phpPlugin)
pluginMgr.Register(dbPlugin)

// External plugins (new)
manifests := discovery.Scan(pluginsDir)
for _, m := range manifests {
    ext := external.NewPlugin(m)
    pluginMgr.Register(ext)
}
```

### GUI

The existing `Manager.Plugins()` method returns status for all registered plugins. External plugins appear automatically with no GUI changes needed.

### Hot Reload

Not in scope. Plugins are loaded at startup only. Restart Flock to pick up new plugins.

## Package Layout

```
internal/
  discovery/          # Scan(), PluginManifest, validation
    discovery.go
    discovery_test.go
  external/           # ExternalPlugin adapter, JSON-RPC client
    plugin.go
    plugin_test.go
    jsonrpc.go        # JSON-RPC 2.0 over stdin/stdout
    jsonrpc_test.go
```

The `plugin` package (interfaces) and `Manager` stay untouched.

## Testing Strategy

1. **Discovery tests** — mock filesystem with valid/invalid plugin directories, verify `Scan()` returns correct manifests and skips bad ones
2. **ExternalPlugin adapter tests** — mock subprocess stdin/stdout with a fake JSON-RPC responder, verify each method sends correct RPC and maps responses/errors
3. **Timeout/crash tests** — verify a hanging or crashing plugin gets marked degraded
4. **Integration test** — build a tiny test plugin executable that speaks JSON-RPC, place in temp directory with `plugin.json`, verify full flow: discovery -> load -> init -> start -> upstream resolution -> stop

## Future

- **flock-sdk** — Go library for plugin authors that handles JSON-RPC boilerplate (see roadmap)
- **Hot reload** — detect new plugins without restart
- **Plugin marketplace** — discovery and installation from a registry
