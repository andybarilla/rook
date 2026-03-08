# Rook Core Design

**Date**: 2026-03-03
**Status**: Approved

## Overview

Rook is a cross-platform, open-source local development environment manager — a Go + Wails desktop app serving as a community alternative to Laravel Herd. It manages local vhosts, SSL, PHP runtimes, and database services via a plugin architecture that can be extended to support any language stack.

**Tech stack**: Go + Wails (webview-based desktop), Caddy embedded as a Go library
**Target platforms**: macOS, Linux, Windows (all three from day one)
**v1 runtime support**: PHP only
**Future**: Plugin API opened to external authors for Node, Quarkus, etc.

---

## Architecture

Three layers:

### Core (always running)
- **Plugin host** — discovers, loads, and manages plugin lifecycle
- **Caddy manager** — embeds Caddy as a Go library; owns all vhost and TLS config; hot-reloads via `caddy.Run()` without subprocess restarts
- **Site registry** — persists known local sites to `~/.config/rook/sites.json`
- **System tray + Wails GUI**

### First-party plugins (bundled, loaded at startup)
- `rook-php` — PHP binary version management, PHP-FPM pool lifecycle, per-site version selection
- `rook-databases` — MySQL, PostgreSQL, Redis service lifecycle
- `rook-ssl` — mkcert integration for local CA trust and per-site/wildcard TLS certs

### Future plugins (same API, not bundled in v1)
- `rook-node`, `rook-quarkus`, and others — external authors use the same plugin interface

The plugin host is the only layer that knows about plugins. Caddy and the site registry are plugin-agnostic.

---

## Plugin Contract

All plugins are in-process Go structs (no IPC in v1). Three interfaces:

```go
type Plugin interface {
    ID() string
    Name() string
    Init(host PluginHost) error
    Start() error
    Stop() error
}

type RuntimePlugin interface {
    Plugin
    UpstreamFor(site Site) (string, error) // e.g. "fastcgi://unix//tmp/php-fpm-8.3.sock"
    Handles(site Site) bool
}

type ServicePlugin interface {
    Plugin
    ServiceStatus() ServiceStatus
    StartService() error
    StopService() error
}
```

`PluginHost` (passed to `Init`) is the core's API surface for plugins — register vhost hooks, emit events, write to the GUI.

First-party plugins are compiled into the binary and registered at startup. Dynamic loading for external plugins is deferred to a future phase.

---

## Site Registry & Caddy Integration

**Site entry** (`~/.config/rook/sites.json`):
```json
{
  "path": "/home/user/code/myapp",
  "domain": "myapp.test",
  "php_version": "8.3",
  "tls": true
}
```

Sites are added via the GUI or a `rook link` CLI subcommand (same binary).

**Caddy config generation**: When the site registry changes, the core generates a `caddy.Config` struct in memory and calls `caddy.Run()` for a zero-downtime hot reload. For each site, the core asks each registered `RuntimePlugin` `Handles(site)` in registration order — the first match provides the upstream string. If no plugin matches, Caddy serves the directory as static files.

---

## Data Flow

**Adding a site:**
1. User picks a directory in the GUI (or runs `rook link`)
2. Core infers domain from directory name (`myapp` → `myapp.test`); user can override
3. `rook-ssl` issues a cert for the domain
4. `RuntimePlugin`s are polled for `Handles(site)` — first match sets upstream
5. Caddy config is regenerated and hot-reloaded
6. Site is live at `https://myapp.test`

**Request path:**
```
Browser → Caddy (443) → upstream resolved by plugin
                          ├─ PHP site   → PHP-FPM unix socket
                          ├─ static     → Caddy file server
                          └─ (future)   → plugin-defined upstream
```

**PHP-FPM pool management**: One pool per PHP version in use. Pools start on demand (first site needing that version) and stay running until Rook quits.

**Startup sequence:**
1. Load config + site registry
2. Init plugins in registration order
3. `rook-ssl` ensures CA is trusted and certs exist for all sites
4. `rook-php` starts FPM pools for all PHP versions referenced by current sites
5. Caddy starts with full generated config
6. GUI becomes ready

---

## Error Handling

- Plugin `Init`/`Start` failures are non-fatal — Rook starts with the plugin marked degraded and shows a GUI warning. A broken database plugin does not affect PHP sites.
- Caddy config errors (missing cert, port conflict) surface as actionable notifications.
- FPM pool failures are per-version — other versions keep running.
- All errors logged to `~/.local/share/rook/rook.log` (XDG on Linux/macOS; `AppData\Roaming\rook` on Windows).

---

## Testing Strategy

All features implemented TDD.

| Layer | Approach |
|---|---|
| Plugin interface | Unit tests with a mock `PluginHost` |
| Site registry | Unit tests: add, remove, persist, load |
| Caddy config generation | Unit tests asserting generated config structs for given site inputs |
| PHP plugin | Integration tests: real FPM pool against a fixture PHP file |
| SSL plugin | Integration test with mkcert in non-interactive mode |
| GUI | Manual + Wails e2e tests for critical paths (add site, toggle service) |

CI matrix: `ubuntu-latest`, `macos-latest`, `windows-latest` on GitHub Actions.
