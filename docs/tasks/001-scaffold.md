# Task 001: Project Scaffold

## Progress Summary

**Status**: Not Started

- [ ] Step 1: Initialize Wails project
- [ ] Step 2: Implement config paths with TDD
- [ ] Step 3: Set up GitHub Actions CI

## Overview

Bootstrap the Flock Go + Wails desktop app. Creates the project structure, establishes platform-aware config/data paths, and sets up a CI matrix across macOS, Linux, and Windows.

## Current State Analysis

- Empty repo with only planning docs (`docs/`, `CLAUDE.md`, `README.md`)
- No Go module, no source code
- Go module path will be `github.com/andybarilla/flock`

## Target State

- Wails v2 project initialized and building (`wails build` succeeds)
- `internal/config` package with `ConfigDir()`, `DataDir()`, `SitesFile()`, `LogFile()` — all platform-aware
- Unit tests for config paths passing
- GitHub Actions CI running `go test ./...` on ubuntu, macos, windows

## Implementation Steps

### Step 1: Initialize Wails project

Run `wails init` to generate the project scaffold, then verify it builds.

**Files created by Wails:**

- `go.mod` — Go module (`github.com/andybarilla/flock`)
- `main.go` — Entry point wiring Wails options
- `app.go` — App struct with `startup`/`shutdown` hooks
- `wails.json` — Wails project config
- `frontend/` — Webview frontend (vanilla JS default)

### Step 2: Implement config paths (TDD)

Write tests first, then implement the `internal/config` package providing platform-correct paths for config and data files.

**Platform conventions:**
- Linux/macOS config: `~/.config/flock/`
- Linux/macOS data/logs: `~/.local/share/flock/`
- Windows config + data: `%APPDATA%\flock\`

**Files to create:**

- `internal/config/paths_test.go` — Tests for each exported function
- `internal/config/paths.go` — `ConfigDir()`, `DataDir()`, `SitesFile()`, `LogFile()`

### Step 3: Set up GitHub Actions CI

CI workflow that runs `go test ./...` on all three platforms.

**Files to create:**

- `.github/workflows/ci.yml` — Matrix: ubuntu-latest, macos-latest, windows-latest

## Acceptance Criteria

### Functional Requirements

- [ ] `wails dev` launches the app window without errors
- [ ] `wails build` produces a binary in `build/bin/`
- [ ] `ConfigDir()` returns correct path per platform
- [ ] `DataDir()` returns correct path per platform
- [ ] `SitesFile()` returns `<ConfigDir>/sites.json`
- [ ] `LogFile()` returns `<DataDir>/flock.log`

### Technical Requirements

- [ ] All tests pass (`go test ./...`)
- [ ] CI passes on all three platforms

## Files Involved

### New Files

- `go.mod`
- `main.go`
- `app.go`
- `wails.json`
- `frontend/` (Wails-generated)
- `internal/config/paths.go`
- `internal/config/paths_test.go`
- `.github/workflows/ci.yml`

## Notes

- Design doc: `docs/plans/2026-03-03-flock-core-design.md`
- Detailed implementation reference: `docs/plans/2026-03-03-flock-core.md` (Task 1)
- Wails v2 requires `libgtk-3-dev libwebkit2gtk-4.0-dev` on Linux (handled in CI)

## Dependencies

- None — this is the first task
