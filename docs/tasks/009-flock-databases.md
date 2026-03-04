# Task 009: flock-databases Plugin

## Progress Summary

**Status**: Not Started

- [ ] Step 1: DBRunner interface & types
- [ ] Step 2: Config loading & persistence
- [ ] Step 3: Plugin core (ServicePlugin)
- [ ] Step 4: ProcessRunner (concrete DBRunner)
- [ ] Step 5: Core integration
- [ ] Step 6: Wails bindings (App)
- [ ] Step 7: GUI — ServiceList component
- [ ] Step 8: Binary detection on Init
- [ ] Step 9: Run all tests & final verification
- [ ] Step 10: Update roadmap

## Overview

Add a unified databases plugin that manages MySQL, PostgreSQL, and Redis processes. The plugin implements `ServicePlugin`, uses a `DBRunner` interface (same pattern as flock-php's `FPMRunner`), and includes a basic GUI services panel with start/stop controls.

## Current State Analysis

- Phase 1 complete: scaffold, registry, plugin host, caddy, ssl, php, core wiring, GUI site list
- Plugin system established with `ServicePlugin` and `RuntimePlugin` interfaces
- `flock-php` provides reference implementation for the runner pattern (`FPMRunner`)
- Core wires plugins via `Config` struct and registers in `NewCore()`
- GUI has Sites section — need to add Services section

## Target State

- `internal/databases/` package with Plugin, DBRunner interface, ProcessRunner, Config
- Plugin registered in Core, exposed via Wails bindings
- GUI "Services" panel showing MySQL, PostgreSQL, Redis with start/stop buttons
- Binary detection on Init (services without binaries shown as "Not installed")
- Config persisted in `~/.config/flock/databases.json`
- ~20 unit tests passing

## Implementation Steps

### Step 1: DBRunner Interface & Types

Create `internal/databases/runner.go` with `ServiceType`, `ServiceConfig`, `ServiceInfo`, `ServiceStatus`, and `DBRunner` interface.

**Files to create:**
- `internal/databases/runner.go` — Interface and types

### Step 2: Config Loading & Persistence

Create config structs and load/save from `databases.json`. TDD — tests first.

**Files to create:**
- `internal/databases/config.go` — Config, SvcConfig, DefaultConfig, LoadConfig, SaveConfig
- `internal/databases/config_test.go` — 4 tests

### Step 3: Plugin Core (ServicePlugin)

Create the Plugin struct implementing ServicePlugin with Init/Start/Stop, plus per-service StartSvc/StopSvc/ServiceStatuses. TDD — tests first.

**Files to create:**
- `internal/databases/databases.go` — Plugin implementation
- `internal/databases/databases_test.go` — ~12 tests with mock DBRunner

### Step 4: ProcessRunner (Concrete DBRunner)

Create the concrete os/exec-based ProcessRunner for starting/stopping MySQL, PostgreSQL, Redis.

**Files to create:**
- `internal/databases/process.go` — ProcessRunner, BinaryFor, CheckBinary
- `internal/databases/process_test.go` — 4 tests

### Step 5: Core Integration

Update Core to register the databases plugin and expose database service methods.

**Files to modify:**
- `internal/core/core.go` — Add DBRunner/DBConfigPath/DBDataRoot to Config, register plugin
- `internal/core/core_test.go` — Add stubDBRunner, update testConfig, add TestPluginsIncludesDatabases

### Step 6: Wails Bindings (App)

Update app.go with DatabaseServices/StartDatabase/StopDatabase methods and wire ProcessRunner.

**Files to modify:**
- `app.go` — Add Wails-bound methods, create ProcessRunner in startup()

### Step 7: GUI — ServiceList Component

Create Svelte component for database services and integrate into App.svelte.

**Files to create:**
- `frontend/src/ServiceList.svelte` — Service table with start/stop buttons

**Files to modify:**
- `frontend/src/App.svelte` — Import ServiceList, add Services section

### Step 8: Binary Detection on Init

Add binary checking during Init to disable services when binaries are missing.

**Files to modify:**
- `internal/databases/databases.go` — Add binaryChecker, check in Init
- `internal/databases/databases_test.go` — Add TestInitDetectsDisabledBinaries

### Step 9: Run All Tests & Final Verification

Run full test suite, go vet, and build verification.

### Step 10: Update Roadmap

Mark flock-databases as complete in `docs/ROADMAP.md`.

## Acceptance Criteria

### Functional Requirements

- [ ] Plugin starts/stops MySQL, PostgreSQL, Redis via system binaries
- [ ] Config persisted in `databases.json` with port, autostart, dataDir per service
- [ ] Missing binaries detected and services shown as "Not installed"
- [ ] Autostart services launch when Flock starts
- [ ] GUI shows service status with start/stop controls
- [ ] Services can be started/stopped independently

### Technical Requirements

- [ ] All tests pass (TDD — tests written before implementation)
- [ ] No breaking changes to existing functionality
- [ ] `go test ./...` passes clean
- [ ] `go vet ./...` passes clean

## Files Involved

### New Files

- `internal/databases/runner.go`
- `internal/databases/config.go`
- `internal/databases/config_test.go`
- `internal/databases/databases.go`
- `internal/databases/databases_test.go`
- `internal/databases/process.go`
- `internal/databases/process_test.go`
- `frontend/src/ServiceList.svelte`

### Modified Files

- `internal/core/core.go`
- `internal/core/core_test.go`
- `app.go`
- `frontend/src/App.svelte`

## Notes

- Design doc: `docs/plans/2026-03-04-flock-databases-design.md`
- Implementation plan: `docs/plans/2026-03-04-flock-databases.md`
- Architecture: `docs/plans/2026-03-03-flock-core-design.md`
- Reference plugin: `internal/php/php.go` (FPMRunner pattern)

## Dependencies

- Task 007 (Core wiring) — complete
- Task 008 (GUI site list) — complete
