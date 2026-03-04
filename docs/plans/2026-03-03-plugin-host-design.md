# Plugin Interfaces + Host Design

**Date**: 2026-03-03
**Status**: Approved

## Overview

Plugin system for Flock: three plugin interfaces (`Plugin`, `RuntimePlugin`, `ServicePlugin`), a `Host` interface plugins receive for accessing core services, and a `Manager` that owns plugin lifecycle and upstream resolution.

## Package

Single `internal/plugin` package. Interfaces, types, and `Manager` all live here. No sub-packages.

## Interfaces

### Plugin (base)

```go
type Plugin interface {
    ID() string
    Name() string
    Init(host Host) error
    Start() error
    Stop() error
}
```

### RuntimePlugin

Extends Plugin with request routing for sites (e.g. PHP-FPM upstream).

```go
type RuntimePlugin interface {
    Plugin
    Handles(site registry.Site) bool
    UpstreamFor(site registry.Site) (string, error)
}
```

### ServicePlugin

Extends Plugin with background service lifecycle (e.g. MySQL, Redis).

```go
type ServicePlugin interface {
    Plugin
    ServiceStatus() ServiceStatus
    StartService() error
    StopService() error
}
```

## Types

```go
type ServiceStatus int
const (
    ServiceStopped ServiceStatus = iota
    ServiceRunning
    ServiceDegraded
)

type PluginStatus int
const (
    PluginReady PluginStatus = iota
    PluginDegraded
)

type PluginInfo struct {
    ID     string
    Name   string
    Status PluginStatus
}
```

## Host Interface

The API surface plugins receive via `Init`. Minimal for now — site access and logging only. Caddy hooks and GUI notifications added in later tasks.

```go
type Host interface {
    Sites() []registry.Site
    GetSite(domain string) (registry.Site, bool)
    Log(pluginID string, msg string, args ...any)
}
```

## SiteSource Interface

Decouples Manager from concrete `registry.Registry`:

```go
type SiteSource interface {
    List() []registry.Site
    Get(domain string) (registry.Site, bool)
}
```

`registry.Registry` already satisfies this interface.

## Manager

Concrete struct that implements `Host` and manages plugin lifecycle.

```go
type Manager struct {
    registry  SiteSource
    logger    *log.Logger
    plugins   []pluginEntry
}

type pluginEntry struct {
    plugin Plugin
    status PluginStatus
}
```

### Methods

- **`Register(p Plugin)`** — adds a plugin (called before Init/Start)
- **`InitAll()`** — calls `Init(host)` on each plugin in registration order; failures log + mark degraded
- **`StartAll()`** — calls `Start()` on non-degraded plugins; failures log + mark degraded
- **`StopAll()`** — calls `Stop()` on all plugins in reverse order; errors logged, all plugins attempted
- **`ResolveUpstream(site) (string, error)`** — iterates RuntimePlugins in registration order, returns first match's `UpstreamFor()`. Returns `("", nil)` if no match (static file serving).
- **`Plugins() []PluginInfo`** — returns ID, name, status for each plugin

### Error Handling

Per core design: plugin Init/Start failures are non-fatal. The host logs the error and marks the plugin as degraded. Other plugins continue normally.

## Testing Strategy

TDD with mock plugins defined in test file:

1. Register + InitAll + StartAll succeeds for healthy plugins
2. InitAll marks a failing plugin as degraded, continues others
3. StartAll skips degraded plugins, marks newly failing ones degraded
4. StopAll calls Stop in reverse order on all plugins
5. ResolveUpstream returns first matching RuntimePlugin's upstream
6. ResolveUpstream returns empty string when no plugin handles site
7. Plugins() returns correct status for each plugin
8. Host.Sites() and Host.GetSite() delegate to SiteSource

## Files

- `internal/plugin/plugin.go` — interfaces and types
- `internal/plugin/manager.go` — Manager implementation
- `internal/plugin/manager_test.go` — all tests
