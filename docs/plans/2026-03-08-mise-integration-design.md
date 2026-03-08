# Mise Integration — Runtime Resolution, Auto-Detection, and Installation

**Date:** 2026-03-08
**Status:** Approved
**Approach:** Shell out to `mise` CLI with graceful degradation to system PATH

## Context

Rook currently assumes language runtimes (PHP, Node.js) are pre-installed on the system. Users must manually install the correct versions and manually specify them when adding sites. [mise](https://mise.jdx.dev) is a polyglot runtime version manager that handles installation and version switching for PHP, Node, Python, Ruby, Go, and hundreds of other tools.

This design integrates mise as an optional runtime backend — using it when available, falling back to current system PATH behavior when not.

## Design Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Integration method | Shell out to `mise` CLI | Stable CLI contract, avoids coupling to internal paths |
| Availability | Graceful degradation | Use mise if present, fall back to system binaries if not |
| Version detection | Auto-detect with override | Scan project config files, pre-fill form, user can change |
| Runtime installation | Explicit user action | Show warning + Install button; no silent downloads |
| Supported runtimes | PHP + Node only | Only runtimes with existing Rook plugins |
| Settings UI | Status indicator only | Show mise detected/not-found; no full runtime manager |

## RuntimeResolver (internal/mise/)

A shared service that plugins call to resolve binary paths.

### Interface

```go
type RuntimeResolver struct{}

func New() *RuntimeResolver

// Availability
func (r *RuntimeResolver) Available() bool

// Binary resolution (falls back to exec.LookPath when mise unavailable)
func (r *RuntimeResolver) Which(tool string) (string, error)
func (r *RuntimeResolver) WhichVersion(tool, version string) (string, error)

// Version detection from project config files (.mise.toml, .tool-versions)
func (r *RuntimeResolver) Detect(siteDir string) (map[string]string, error)

// Installation (requires mise)
func (r *RuntimeResolver) Install(tool, version string) error
func (r *RuntimeResolver) IsInstalled(tool, version string) bool
func (r *RuntimeResolver) ListInstalled(tool string) ([]string, error)
```

### Fallback Behavior

- `Available()` — checks if `mise` is in PATH (cached on first call)
- `Which(tool)` / `WhichVersion(tool, version)` — calls `mise which <tool>`, falls back to `exec.LookPath(tool)` if mise unavailable
- `Detect(siteDir)` — parses `.mise.toml` or `.tool-versions` in the directory; returns empty map if mise unavailable or no config found
- `Install()` / `ListInstalled()` — return error if mise unavailable

## Plugin Integration

The `RuntimeResolver` is created in `internal/core/` during startup and passed to PHP and Node plugins via dependency injection.

### PHP Plugin Changes

- Replace assumption of `php-fpm{version}` in PATH with `resolver.WhichVersion("php", version)` call
- Use resolved binary path when starting FPM pools

### Node Plugin Changes

- Replace bare `npm start` with version-aware resolution via `resolver.WhichVersion("node", version)`
- Set resolved binary's directory on the process PATH so npm resolves correctly

### No Interface Changes

The `RuntimePlugin` interface is unchanged. The resolver is an internal implementation detail within each plugin.

## Auto-Detection on Site Add

### Flow

1. User enters a project path in the Add Site modal
2. Frontend calls `DetectSiteVersions(path)` (debounced on path input change)
3. Backend calls `resolver.Detect(path)` which parses `.mise.toml` / `.tool-versions`
4. Frontend pre-fills PHP Version and Node Version fields with detected values
5. Subtle "detected from .mise.toml" hint shown next to populated fields
6. User can override values before submitting

### Backend API

```go
func (a *App) DetectSiteVersions(path string) (map[string]string, error)
```

### Frontend Behavior

- Debounced call on path input change
- Populate fields + show detection hint if versions found
- Fields stay empty if no config file or mise unavailable (current behavior)

## Missing Runtime Warning + Install

### When a required runtime isn't found

After a site is added or on app startup:

1. Site appears with a warning badge: "PHP 8.3 not found"
2. If mise is available, badge includes an "Install" button
3. User clicks Install → backend runs `mise install php@8.3`
4. Loading spinner during install, success/error toast on completion
5. Warning badge disappears on success

### Backend API

```go
func (a *App) InstallRuntime(tool, version string) error
func (a *App) CheckRuntimes() ([]RuntimeStatus, error)
```

`RuntimeStatus` contains: tool name, version, whether installed, site domain.

### Frontend Behavior

- `CheckRuntimes()` called on mount and after site add
- Warning badges shown on SiteCard and in table view for missing runtimes
- Install button only shown when mise is available
- Loading state during installation with toast feedback

## Settings — Mise Status

A "Runtime Manager" card in the Settings tab:

- **When mise is detected:** Green badge + version string (e.g., "mise 2024.12.0")
- **When mise is not found:** Gray badge + text "Install mise for automatic runtime version management" + link to mise.jdx.dev

### Backend API

```go
func (a *App) MiseStatus() (MiseInfo, error)
```

`MiseInfo` contains: `Available bool`, `Version string`.

No runtime listing or version management UI in Settings.

## What's Not Included

- Python, Ruby, Go runtime support (no Rook plugins for these yet)
- Full runtime manager UI in Settings (premature without more plugins)
- Auto-installation of runtimes (user must click Install)
- Bundling/embedding mise in Rook (users install it themselves)
- mise configuration management (Rook reads but doesn't write mise config)
