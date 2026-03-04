# Task 011: flock-node Plugin

## Progress Summary

**Status**: Not Started

- [ ] Step 1: Add NodeVersion to Site struct
- [ ] Step 2: Node plugin — NodeRunner interface and Plugin struct
- [ ] Step 3: ProcessRunner concrete implementation
- [ ] Step 4: Wire node plugin into Core
- [ ] Step 5: Update AddSite to accept NodeVersion
- [ ] Step 6: Final verification & cleanup

## Overview

Add a built-in Node.js runtime plugin that manages per-site `npm start` processes with auto-assigned ports and HTTP reverse proxy upstream. Mirrors flock-php's pattern — a `NodeRunner` interface injected into a `Plugin` struct implementing `RuntimePlugin` + `ServicePlugin`. Each Node-enabled site gets a process on port 3100+. Caddy reverse-proxies HTTP traffic to it.

## Current State Analysis

- Phase 1 and Phase 2 complete: all built-in plugins (SSL, PHP, databases) work
- Phase 3 plugin discovery complete — external plugins can be loaded
- `registry.Site` has `PHPVersion` but no `NodeVersion` field
- No Node.js runtime support exists
- PHP plugin (`internal/php/`) provides the pattern to follow: interface injection, mock-friendly testing

## Target State

- `registry.Site` gains a `NodeVersion` field (e.g., `"system"`)
- `internal/node/` package with `NodeRunner` interface, `Plugin` struct, and `ProcessRunner`
- Node plugin registered in Core alongside existing plugins
- `AddSite` API accepts `nodeVersion` parameter
- Frontend form includes Node Version field
- ~14 unit tests + 4 integration tests passing

## Implementation Steps

### Step 1: Add NodeVersion to Site struct

Add `NodeVersion string` field to `registry.Site` with JSON tag `node_version,omitempty`. Write persistence test.

**Files to modify:**
- `internal/registry/site.go` — Add NodeVersion field
- `internal/registry/registry_test.go` — Add TestNodeVersionPersistence

### Step 2: Node plugin — NodeRunner interface and Plugin struct

Create `internal/node/` package with `NodeRunner` interface (StartApp, StopApp, IsRunning, AppPort) and `Plugin` struct implementing `RuntimePlugin` + `ServicePlugin`. Uses mock-based testing.

**Files to create:**
- `internal/node/node.go` — NodeRunner interface, Plugin struct
- `internal/node/node_test.go` — 10 tests: ID/Name, Handles, Start (multiple scenarios), Stop, UpstreamFor, ServiceStatus

### Step 3: ProcessRunner concrete implementation

Create `ProcessRunner` using `os/exec` to manage `npm start` processes with PORT env var. Goroutine cleanup on process exit.

**Files to create:**
- `internal/node/process.go` — ProcessRunner struct using os/exec
- `internal/node/process_test.go` — 4 tests: start/stop lifecycle, stop nonexistent, IsRunning default, AppPort default

### Step 4: Wire node plugin into Core

Add `NodeRunner` to `core.Config`, create and register `node.Plugin` in `NewCore()` between PHP and databases plugins. Update `app.go` to instantiate `ProcessRunner`.

**Files to modify:**
- `internal/core/core.go` — Add NodeRunner to Config, register node plugin
- `internal/core/core_test.go` — Add stubNodeRunner, update testConfig, add TestNodePluginStartsForNodeSites, update plugin counts
- `app.go` — Add node import and ProcessRunner to config

### Step 5: Update AddSite to accept NodeVersion

Update `AddSite` signature in `app.go` and frontend bindings/form to include nodeVersion parameter.

**Files to modify:**
- `app.go` — Add nodeVersion parameter to AddSite
- `frontend/src/App.svelte` — Update handleAdd signature
- `frontend/src/AddSiteForm.svelte` — Add nodeVersion field and input
- `frontend/wailsjs/go/main/App.js` — Update AddSite binding
- `frontend/wailsjs/go/main/App.d.ts` — Update AddSite type

### Step 6: Final Verification & Cleanup

Run full test suite, create task file, update roadmap.

**Files to create/modify:**
- `docs/tasks/011-flock-node.md` — This file
- `docs/ROADMAP.md` — Mark flock-node as complete (after PR merge)

## Acceptance Criteria

### Functional Requirements

- [ ] Sites with `node_version` set get an `npm start` process on port 3100+
- [ ] Ports assigned sequentially, sorted by domain
- [ ] App failures logged and skipped (non-fatal)
- [ ] Node plugin provides HTTP upstream for Caddy reverse proxy
- [ ] AddSite accepts nodeVersion parameter
- [ ] Frontend form includes Node Version input

### Technical Requirements

- [ ] All tests pass (TDD — tests written before implementation)
- [ ] No breaking changes to existing functionality
- [ ] `go test ./...` passes clean
- [ ] No new external dependencies (stdlib only)

## Files Involved

### New Files

- `internal/node/node.go`
- `internal/node/node_test.go`
- `internal/node/process.go`
- `internal/node/process_test.go`

### Modified Files

- `internal/registry/site.go`
- `internal/registry/registry_test.go`
- `internal/core/core.go`
- `internal/core/core_test.go`
- `app.go`
- `frontend/src/App.svelte`
- `frontend/src/AddSiteForm.svelte`
- `frontend/wailsjs/go/main/App.js`
- `frontend/wailsjs/go/main/App.d.ts`

## Notes

- Design doc: `docs/plans/2026-03-04-flock-node-design.md`
- Implementation plan: `docs/plans/2026-03-04-flock-node-plan.md`
- Architecture: `docs/plans/2026-03-03-flock-core-design.md`
- Reference pattern: `internal/php/` (FPMRunner interface injection)

## Dependencies

- Task 010 (Plugin discovery) — complete
- Task 007 (Core wiring) — complete
