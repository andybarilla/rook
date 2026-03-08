# Rename Rook to Rook — Design

## Summary

Rename the project from "Rook" to "Rook" across the entire codebase, infrastructure, and public presence. "Rook" is a type of crow — short, memorable, easy to type, and avoids the naming conflict with `/usr/bin/rook` (the POSIX file-locking utility from util-linux).

## New Identity

- **Project name**: Rook
- **CLI command**: `rook`
- **Domain**: getrook.dev
- **Plugin prefix**: `rook-` (rook-php, rook-node, rook-ssl, rook-databases)
- **Go module**: `github.com/andybarilla/rook`
- **GitHub repo**: `andybarilla/rook` (manual rename by owner)

## Scope of Changes

### Code (Go)

- Rename Go module in `go.mod` from `github.com/andybarilla/rook` to `github.com/andybarilla/rook`
- Update all import paths across ~27 Go files (~112 occurrences)
- Rename plugin identifiers: `rook-ssl` → `rook-ssl`, `rook-php` → `rook-php`, `rook-node` → `rook-node`, `rook-databases` → `rook-databases`
- Update config directory paths (e.g., `~/.rook/` → `~/.rook/`)
- Update any hardcoded binary names or app identifiers

### Configuration Files

- `wails.json`: update app name and references
- `.github/workflows/release.yml`: update binary/artifact names
- `CLAUDE.md`, `README.md`, `LEARNINGS.md`: update all references

### Documentation

- `docs/ROADMAP.md`: update project description and plugin names
- `docs/tasks/*.md`: update references
- `docs/plans/*.md`: update references (these are historical, so light-touch)

### Frontend

- Update any UI text that says "Rook" to "Rook"
- Update window titles, tray menu labels, about text

### Static Site

- Rebuild for getrook.dev (separate task, after core rename)

### Infrastructure (Manual)

- Rename GitHub repo from `andybarilla/rook` to `andybarilla/rook`
- Update DNS / domain to getrook.dev
- Update any CI secrets or deployment configs referencing the old name

## Out of Scope

- The static site redesign is a separate follow-up task
- GitHub repo rename is a manual step by the owner

## Migration Notes

- Historical docs/plans can be updated with a simple find-and-replace; no need to preserve the old name in historical context
- The `rook-` plugin prefix in external plugin names (discovery, manifests) must be updated to `rook-`
