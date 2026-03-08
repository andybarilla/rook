# rook-node Plugin Design

## Overview

A built-in plugin for managing Node.js development servers. Mirrors the rook-php pattern — implements both `RuntimePlugin` and `ServicePlugin`. Each Node-enabled site gets its own `npm start` process on an auto-assigned port, with Caddy reverse-proxying HTTP traffic to it.

## Site Registry Changes

Add `NodeVersion string` to `registry.Site`:

```go
type Site struct {
    Path        string `json:"path"`
    Domain      string `json:"domain"`
    PHPVersion  string `json:"php_version,omitempty"`
    NodeVersion string `json:"node_version,omitempty"`
    TLS         bool   `json:"tls"`
}
```

When `NodeVersion` is non-empty (e.g., `"system"`), the site is Node-enabled. Stored in `sites.json`.

## Package Structure

```
internal/node/
  node.go         — Plugin struct + NodeRunner interface
  node_test.go    — Tests with mockNodeRunner + mockHost
  process.go      — ProcessRunner (concrete impl using os/exec)
  process_test.go
```

## NodeRunner Interface

```go
type NodeRunner interface {
    StartApp(siteDir string, port int) error
    StopApp(siteDir string) error
    IsRunning(siteDir string) bool
    AppPort(siteDir string) int
}
```

## Plugin Struct

```go
type Plugin struct {
    runner   NodeRunner
    host     plugin.Host
    basePort int               // 3100
    portMap  map[string]int    // domain -> assigned port
}
```

## Key Behaviors

### Handles(site)

Returns `site.NodeVersion != ""`.

### UpstreamFor(site)

Returns `fmt.Sprintf("http://127.0.0.1:%d", portMap[site.Domain])`. Unlike PHP (FastCGI unix socket), Node uses plain HTTP — Caddy handles this natively via standard reverse proxy.

### Start()

1. Iterates `host.Sites()`, filters to Node-enabled sites
2. Sorts by domain for deterministic port assignment
3. Assigns ports sequentially from base port (3100, 3101, ...)
4. Calls `runner.StartApp(site.Path, port)` for each
5. Logs and skips on individual failure (non-fatal)

### Stop()

Stops all running Node apps via `runner.StopApp()`.

### ServicePlugin methods

`StartService()`/`StopService()` delegate to `Start()`/`Stop()`. `ServiceStatus()` reports running/stopped based on runner state.

## ProcessRunner (Concrete Implementation)

Uses `os/exec.Command("npm", "start")` with:
- Working directory: site path
- Environment: `PORT=<assigned_port>`
- Stores `*exec.Cmd` handles in a map keyed by site directory

No process supervision — if an app crashes, it stays down until manually restarted from the GUI or Rook restarts.

## Port Assignment

- Base port: 3100
- Sequential assignment sorted by domain name
- Stable across restarts if site list unchanged
- App reads `PORT` env var (standard Node.js convention)

## Core Wiring

### core.Config

```go
type Config struct {
    // ... existing fields ...
    NodeRunner node.NodeRunner
}
```

### core.go — Plugin registration

```go
nodePlugin := node.NewPlugin(cfg.NodeRunner)
pluginMgr.Register(sslPlugin)
pluginMgr.Register(phpPlugin)
pluginMgr.Register(nodePlugin)
pluginMgr.Register(dbPlugin)
```

### app.go

Instantiate `node.NewProcessRunner()` and pass to `core.Config.NodeRunner`.

## Caddy Integration

No Caddy changes needed. `ResolveUpstream(site)` returns the first matching plugin's upstream. For Node sites, the upstream is `http://127.0.0.1:<port>` — a valid Caddy reverse proxy target.

## Testing Strategy

TDD with mock runner and host, mirroring rook-php test patterns.

**Mock types:**
- `mockNodeRunner` — tracks calls, configurable errors, reports running state
- `mockHost` — returns configured site list

**Test cases:**
1. Init stores host reference
2. Start starts apps for Node-enabled sites only
3. Start skips sites without NodeVersion
4. Start assigns sequential ports from base port
5. Start logs and continues on individual app failure
6. Stop stops all running apps
7. Handles returns true when NodeVersion set, false otherwise
8. UpstreamFor returns correct HTTP upstream for site
9. UpstreamFor returns error for unknown site
10. ServiceStatus/StartService/StopService delegate correctly

## Decisions

- **Built-in plugin** (compiled into binary, not external JSON-RPC)
- **System node only** (no nvm/fnm version management)
- **npm start only** (no configurable start commands)
- **No process supervision** (can add health checks later if needed)
