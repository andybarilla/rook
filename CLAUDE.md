# CLAUDE.md

## Project Overview

Rook is a local development workspace manager written in Go. It allocates ports globally across workspaces, generates environment files from templates, and orchestrates service subsets via profiles.

## Tech Stack

- Go 1.22+, Cobra (CLI), gopkg.in/yaml.v3
- GUI (Wails + React) is planned but not yet implemented

## Architecture

Interface-driven design. Core library in `internal/`, consumed by CLI in `cmd/rook/`.

- `internal/workspace/` — types (`Service`, `Workspace`, `Manifest`) and YAML parsing
- `internal/ports/` — `PortAllocator` interface, `FileAllocator` (JSON-backed)
- `internal/registry/` — `Registry` interface, `FileRegistry` (JSON-backed)
- `internal/profile/` — `Resolve()` expands profiles into service lists
- `internal/envgen/` — Go template resolution for `{{.Host.x}}`/`{{.Port.x}}`
- `internal/health/` — `Check` type with HTTP, TCP, command variants
- `internal/runner/` — `Runner` interface, `ProcessRunner`, `DockerRunner`
- `internal/orchestrator/` — `TopoSort()` + `Orchestrator` (incremental start/stop)
- `internal/discovery/` — `Discoverer` interface for compose, mise, devcontainer
- `internal/cli/` — Cobra commands

## Commands

```bash
go test ./...           # Run all tests (should always pass)
go test ./internal/X/   # Test a single package
go build ./cmd/rook     # Build the binary
go run ./cmd/rook       # Run without building
```

## Development Conventions

- **TDD**: Write tests first, verify they fail, then implement
- **Interface-driven**: Define interfaces, implement against them
- **One responsibility per file**: Each file has a clear purpose
- **No external test frameworks**: Use stdlib `testing` package only
- **Commit style**: `feat:`, `fix:`, `test:`, `chore:` prefixes
- **Test files**: `_test.go` in same package (black-box: `package x_test`)

## Key Patterns

- File-backed persistence: `ports.json` and `workspaces.json` in `$XDG_CONFIG_HOME/rook/`
- Services are either containers (`image` field) or processes (`command` field)
- Port allocation: preferred ports tried first, then sequential from range 10000-60000
- Pinned ports (`pin_port`) error on conflict instead of reassigning
- Profile resolution: entries can be service names, group names, or `*` wildcard
- Template vars: `{{.Host.x}}` resolves to `localhost` (or service name in devcontainer), `{{.Port.x}}` resolves to allocated port

## What's Not Yet Implemented

- `up`, `down`, `restart`, `status`, `logs` CLI commands are stubs (need daemon/orchestrator wiring)
- GUI (Wails desktop app)
- File watching / live reload
