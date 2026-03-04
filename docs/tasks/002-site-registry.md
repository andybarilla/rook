# Task 002: Site Registry

## Progress Summary

**Status**: Complete

- [x] Step 1: Write failing tests
- [x] Step 2: Implement Site type and event types
- [x] Step 3: Implement Registry

## Overview

Implement the site registry — an in-memory + JSON-persisted store of local development sites. Provides CRUD operations, domain lookup, path validation, domain inference, and change notifications via callbacks.

## Current State Analysis

- Task 001 complete: Go + Wails scaffold with `internal/config` package providing `SitesFile()` path
- No `internal/registry` package exists yet
- Architecture design approved in `docs/plans/2026-03-03-flock-core-design.md`
- Detailed design in `docs/plans/2026-03-03-site-registry-design.md`

## Target State

- `internal/registry` package with `Site`, `ChangeEvent`, `Registry` types
- Full API: `New`, `Load`, `List`, `Get`, `Add`, `Update`, `Remove`, `OnChange`, `InferDomain`
- Path validation on `Add` (directory must exist)
- Change notifications fire after successful persistence
- ~12 unit tests passing

## Implementation Steps

### Step 1: Write failing tests

Create `internal/registry/registry_test.go` with all ~12 tests covering:
- Add/List, duplicate domain error, nonexistent path error
- Get (found and not found)
- Update (success and not found)
- Remove (success and not found)
- Persistence (save + reload via new Registry instance)
- OnChange fires with correct event types for Add/Remove/Update
- InferDomain cases

**Files to create:**

- `internal/registry/registry_test.go` — All tests

### Step 2: Implement Site type and event types

Create the `Site` struct, `EventType` constants, and `ChangeEvent` struct.

**Files to create:**

- `internal/registry/site.go` — `Site`, `EventType`, `ChangeEvent`

### Step 3: Implement Registry

Implement the `Registry` struct with all methods: `New`, `Load`, `List`, `Get`, `Add`, `Update`, `Remove`, `OnChange`, `save`, and the package-level `InferDomain` function.

**Files to create:**

- `internal/registry/registry.go` — `Registry` and `InferDomain`

## Acceptance Criteria

### Functional Requirements

- [ ] `Add` persists a site and it appears in `List`
- [ ] `Add` rejects duplicate domains with an error
- [ ] `Add` rejects nonexistent paths with an error
- [ ] `Get` returns a site by domain (and false for unknown domains)
- [ ] `Update` modifies an existing site's properties
- [ ] `Remove` deletes a site by domain
- [ ] `OnChange` listeners fire with correct `ChangeEvent` for each mutation
- [ ] `InferDomain` derives `name.test` from directory path
- [ ] Data survives save/reload cycle (JSON persistence)

### Technical Requirements

- [ ] All tests pass (TDD — tests written before implementation)
- [ ] No breaking changes to existing functionality
- [ ] `go test ./...` passes clean

## Files Involved

### New Files

- `internal/registry/site.go`
- `internal/registry/registry.go`
- `internal/registry/registry_test.go`

## Notes

- Design doc: `docs/plans/2026-03-03-site-registry-design.md`
- Architecture: `docs/plans/2026-03-03-flock-core-design.md`
- Implementation reference: `docs/plans/2026-03-03-flock-core.md` (Task 2)
- No concurrency — registry is single-goroutine (Wails main thread)

## Dependencies

- Task 001 (scaffold) — complete
