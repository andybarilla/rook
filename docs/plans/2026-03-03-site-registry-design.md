# Site Registry Design

**Date**: 2026-03-03
**Status**: Approved

## Overview

The site registry persists and manages the list of local development sites. It is the source of truth for which directories are registered as dev sites, their domains, and per-site configuration. Other core components (Caddy manager, plugins) observe registry changes via callbacks.

## Data Model

```go
type Site struct {
    Path       string `json:"path"`
    Domain     string `json:"domain"`
    PHPVersion string `json:"php_version,omitempty"`
    TLS        bool   `json:"tls"`
}
```

Stored as a JSON array in `~/.config/rook/sites.json` (platform-aware via `config.SitesFile()`).

## Change Events

```go
type EventType int

const (
    SiteAdded EventType = iota
    SiteRemoved
    SiteUpdated
)

type ChangeEvent struct {
    Type    EventType
    Site    Site
    OldSite *Site // non-nil for SiteUpdated
}
```

## API

| Method | Signature | Behavior |
|---|---|---|
| New | `New(path string) *Registry` | Create registry backed by JSON file |
| Load | `Load() error` | Read from disk; missing file = empty |
| List | `List() []Site` | Return copy of all sites |
| Get | `Get(domain string) (Site, bool)` | Lookup single site by domain |
| Add | `Add(site Site) error` | Validate path exists, reject duplicate domain, save, notify |
| Update | `Update(domain string, fn func(*Site)) error` | Find by domain, apply mutation, save, notify with old+new |
| Remove | `Remove(domain string) error` | Remove by domain, save, notify |
| OnChange | `OnChange(fn func(ChangeEvent))` | Register change listener |
| InferDomain | `InferDomain(path string) string` | Directory name → `name.test` (package-level function) |

### Key decisions

- **Update uses a mutation function** `func(*Site)` — callers modify in-place, avoids partial-update ambiguity.
- **Path validation on Add** — `os.Stat(site.Path)` must succeed and be a directory.
- **Notifications fire after save** — listeners never see uncommitted state.
- **Synchronous notifications** — no goroutines needed; Wails runs on the main thread.

## Persistence

- `save()`: `MkdirAll` parent dir → `json.MarshalIndent` → `os.WriteFile`
- `Load()`: `os.ErrNotExist` treated as empty (fresh installs just work)

## Error Handling

- `Add`: error if directory doesn't exist or domain is duplicate
- `Remove`/`Update`: error if domain not found
- `Load`: error only on read/parse failures (missing file is OK)

## Testing

All TDD. ~12 tests:

1. Add and List
2. Add duplicate domain → error
3. Add nonexistent path → error
4. Get existing domain → found
5. Get nonexistent domain → not found
6. Update existing site
7. Update nonexistent domain → error
8. Remove
9. Remove nonexistent → error
10. Persistence (save + reload)
11. OnChange fires for Add/Remove/Update with correct event types
12. InferDomain cases

## References

- Architecture: `docs/plans/2026-03-03-rook-core-design.md` (Site Registry & Caddy Integration section)
- Implementation plan: `docs/plans/2026-03-03-rook-core.md` (Task 2)
