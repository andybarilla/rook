# rook-php Plugin Design

## Goal

Provide PHP-FPM pool management for Rook-managed sites. Sites with a `PHPVersion` set get routed through PHP-FPM via Caddy's reverse proxy.

## Architecture

Single `internal/php` package implementing both `plugin.RuntimePlugin` (request routing) and `plugin.ServicePlugin` (pool lifecycle). An `FPMRunner` interface abstracts FPM process management for testability.

### Interfaces

**`FPMRunner`** (in `internal/php/php.go`):

```go
type FPMRunner interface {
    StartPool(version string) error
    StopPool(version string) error
    PoolSocket(version string) string
}
```

### Plugin Behavior

**`php.Plugin`** — implements `plugin.RuntimePlugin` + `plugin.ServicePlugin`:

- `ID()` → `"rook-php"`
- `Name()` → `"Rook PHP"`
- `Init(host)` — stores host reference
- `Start()` — scans `host.Sites()`, collects unique non-empty `PHPVersion` values, calls `runner.StartPool(version)` for each. Tracks running pools.
- `Stop()` — stops all running pools via `runner.StopPool(version)`
- `Handles(site)` — returns `site.PHPVersion != ""`
- `UpstreamFor(site)` — returns `"unix/" + runner.PoolSocket(site.PHPVersion)`. Errors if pool not running.
- `ServiceStatus()` — Running if any pools active, Stopped otherwise
- `StartService()` / `StopService()` — delegates to `Start()` / `Stop()`

### Data Flow

```
Site added with PHPVersion="8.3"
  → php.Plugin.Start() collects unique versions, starts FPM pool for 8.3
  → Caddy BuildConfig calls manager.ResolveUpstream(site)
  → manager iterates plugins, php.Plugin.Handles(site) returns true
  → php.Plugin.UpstreamFor(site) returns "unix//tmp/php-fpm-8.3.sock"
  → Caddy reverse_proxy routes requests to FPM socket
```

### Error Handling

- Pool start failure → log warning, skip version (don't block other pools)
- `UpstreamFor()` for non-running pool → returns error (Caddy falls back to file_server)
- Plugin init/start failure → marked degraded by plugin manager, Rook continues

### Scope

- FPMRunner interface + mock for tests (real FPM process management deferred to Core wiring)
- Trust `site.PHPVersion` as-is (no version detection/validation in this task)
