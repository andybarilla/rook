# Caddy Manager Design

**Date**: 2026-03-03
**Status**: Approved

## Overview

Embedded Caddy manager for Rook: generates JSON config from site registry entries and plugin upstream resolution, manages Caddy lifecycle via `caddy.Run()` for zero-downtime hot-reloads.

## Package

Single `internal/caddy` package. Runner interface, Manager struct, and config builder all live here.

## Interfaces

### CaddyRunner

Abstracts Caddy's `Run`/`Stop` for testability. Production implementation wraps real `caddy.Run()` and `caddy.Stop()`.

```go
type CaddyRunner interface {
    Run(cfgJSON []byte) error
    Stop() error
}
```

### UpstreamResolver

Provides upstream lookups for sites. Satisfied by `plugin.Manager.ResolveUpstream`.

```go
type UpstreamResolver interface {
    ResolveUpstream(site registry.Site) (string, error)
}
```

## Manager

```go
type Manager struct {
    runner   CaddyRunner
    resolver UpstreamResolver
    running  bool
}

func NewManager(runner CaddyRunner, resolver UpstreamResolver) *Manager
```

### Methods

- **`Start(sites []registry.Site) error`** — builds config via `BuildConfig`, calls `runner.Run()`, sets `running = true`
- **`Reload(sites []registry.Site) error`** — same as Start; Caddy's `Run` handles hot-reload transparently
- **`Stop() error`** — calls `runner.Stop()`, sets `running = false`

## Config Generation

`BuildConfig(sites []registry.Site, resolver UpstreamResolver) ([]byte, error)` produces Caddy JSON config:

- Admin API disabled (`admin.disabled = true`)
- One HTTP server with listeners on `:80` and `:443`
- One route per site, matched by `host` on `site.Domain`
- For each site: calls `resolver.ResolveUpstream(site)`
  - Non-empty upstream string → `reverse_proxy` handler with `dial` set to the upstream
  - Empty upstream → `file_server` handler with `root` set to `site.Path`
- Returns marshaled JSON bytes

### Generated Config Structure

```json
{
  "admin": {"disabled": true},
  "apps": {
    "http": {
      "servers": {
        "rook": {
          "listen": [":80", ":443"],
          "routes": [
            {
              "match": [{"host": ["myapp.test"]}],
              "handle": [{"handler": "reverse_proxy", "upstreams": [{"dial": "unix//tmp/php-fpm.sock"}]}]
            },
            {
              "match": [{"host": ["static.test"]}],
              "handle": [{"handler": "file_server", "root": "/home/user/static"}]
            }
          ]
        }
      }
    }
  }
}
```

### TLS

Not handled in this task. The rook-ssl plugin (later task) will add TLS automation config. For now, sites listen on both `:80` and `:443` but without cert provisioning — effectively HTTP only until rook-ssl is implemented.

## Testing Strategy

TDD with mock CaddyRunner and mock UpstreamResolver:

1. BuildConfig with static site (no upstream) → file_server handler with correct root
2. BuildConfig with proxied site (upstream returned) → reverse_proxy handler with correct dial
3. BuildConfig with mixed sites → correct routes for each
4. BuildConfig sets admin.disabled = true
5. Start calls runner.Run() with valid JSON
6. Reload calls runner.Run() again (hot-reload)
7. Stop calls runner.Stop()

Tests parse generated JSON to assert structure, no string matching.

## Files

- `internal/caddy/caddy.go` — CaddyRunner interface, UpstreamResolver interface, Manager struct, BuildConfig function
- `internal/caddy/caddy_test.go` — all tests
