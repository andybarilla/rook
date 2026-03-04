# GUI: Site List Design

## Goal

Replace the Wails scaffold demo with a functional site management GUI — list sites, add/remove sites, auto-refresh on changes.

## Architecture

Svelte 3 frontend (already scaffolded) with three components talking to Go backend via Wails auto-generated bindings. No new dependencies.

### Components

**`App.svelte`** — Root layout and state:
- Loads sites on mount via `ListSites()` binding
- Holds reactive `sites` array
- `refreshSites()` re-fetches after add/remove
- Renders header, `SiteList`, `AddSiteForm`
- Shows inline error messages on failure

**`SiteList.svelte`** — Site table:
- Props: `sites`, `onRemove` callback
- Columns: Domain, Path, PHP Version, TLS
- Remove button per row
- Empty state message when no sites

**`AddSiteForm.svelte`** — Add site form:
- Fields: Path, Domain (auto-inferred from path), PHP Version (optional), TLS checkbox
- Submit calls `AddSite` binding via parent callback
- Clears on success, shows error on failure

### Data Flow

```
User adds site → AddSiteForm.onAdd(site)
  → App calls AddSite(path, domain, phpVersion, tls) binding
  → Go: Core.AddSite → Registry.Add → OnChange → Caddy.Reload
  → App calls refreshSites() → ListSites() → updates reactive state
  → SiteList re-renders with new site
```

### Go Backend

Already complete from Core wiring (task 007):
- `ListSites() []registry.Site`
- `AddSite(path, domain, phpVersion string, tls bool) error`
- `RemoveSite(domain string) error`

Wails auto-generates JS bindings when `wails dev` or `wails build` runs.

### Styling

Extends existing dark theme in `style.css`. Table and form styled with vanilla CSS.

### Scope

- Replace Greet demo with site management UI
- Three Svelte components (App, SiteList, AddSiteForm)
- No system tray (deferred)
- No plugin status panel (YAGNI)
- No new dependencies
