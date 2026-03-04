# Core Wiring Design

## Goal

Wire together Registry, Plugin Manager, Caddy Manager, SSL Plugin, and PHP Plugin into a running application with a testable `Core` struct and thin Wails `App` layer.

## Architecture

Single `internal/core/core.go` package containing a `Core` struct that orchestrates the full lifecycle. Dependencies injected via a `Config` struct. `App` (root `app.go`) holds a `Core` reference and exposes Wails-bound methods.

### Core Struct

```go
type Config struct {
    SitesFile   string
    Logger      *log.Logger
    CaddyRunner caddy.CaddyRunner
    FPMRunner   php.FPMRunner
    CertStore   ssl.CertStore
}

type Core struct {
    registry  *registry.Registry
    pluginMgr *plugin.Manager
    caddyMgr  *caddy.Manager
    sslPlugin *ssl.Plugin
    phpPlugin *php.Plugin
    logger    *log.Logger
}
```

### Lifecycle

- `NewCore(cfg Config)` — creates Registry, plugins, Plugin Manager (registers plugins), Caddy Manager, wires registry OnChange listener
- `Start()` — loads registry from disk, calls `pluginMgr.InitAll()`, `pluginMgr.StartAll()`, starts Caddy with initial site config
- `Stop()` — stops Caddy, calls `pluginMgr.StopAll()`
- `Sites()`, `AddSite()`, `RemoveSite()`, `Plugins()` — delegating API for App/GUI

### Data Flow

```
App.startup() → NewCore(cfg) → Core.Start()
  → Registry.Load()
  → PluginMgr.InitAll() (SSL installs CA, PHP stores host)
  → PluginMgr.StartAll() (SSL generates certs, PHP starts pools)
  → CaddyMgr.Start(sites)

Registry.Add/Update/Remove → OnChange → Core.reload()
  → CaddyMgr.Reload(registry.List())

App shutdown → Core.Stop()
  → CaddyMgr.Stop()
  → PluginMgr.StopAll() (PHP stops pools)
```

### App Integration

`app.go` creates `Core` in `startup()` with real or stub dependencies. Wails-bound methods on `App` delegate to `Core`:

- `AddSite(path, domain, phpVersion string, tls bool)` → `core.AddSite()`
- `RemoveSite(domain string)` → `core.RemoveSite()`
- `ListSites() []registry.Site` → `core.Sites()`
- `ListPlugins() []plugin.PluginInfo` → `core.Plugins()`

### Stub Implementations

For this task, real process management is deferred:

- `LoggingCaddyRunner` — logs Run/Stop, no real Caddy embed
- `LoggingFPMRunner` — logs StartPool/StopPool, returns socket paths
- `LocalCertStore` — already exists in `internal/ssl/certstore.go`

### Error Handling

- Plugin init/start failures → non-fatal (handled by plugin manager, logged)
- Caddy start failure → `Core.Start()` returns error
- Registry load failure → `Core.Start()` returns error (creates empty registry if file missing)
- Change listener reload failure → logged, not fatal

### Scope

- `Core` struct with full lifecycle (NewCore, Start, Stop)
- Delegating API methods (Sites, AddSite, RemoveSite, Plugins)
- Registry change listener wired to Caddy reload
- App integration (startup/shutdown)
- Stub CaddyRunner and FPMRunner
- Tests for Core lifecycle, change-driven reload, error cases
