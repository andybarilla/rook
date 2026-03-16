# CLAUDE.md

## Project Overview

Rook is a local development workspace manager written in Go. It allocates ports globally across workspaces, generates environment files from templates, orchestrates service subsets via profiles, and auto-discovers services from docker-compose files. It supports both Docker and Podman (auto-detected, podman preferred).

## Tech Stack

- Go 1.22+, Cobra (CLI), gopkg.in/yaml.v3
- GUI: Wails v2 + React 19 + TypeScript + Tailwind CSS v4 + shadcn/ui (built, at `cmd/rook-gui/`)
- Container runtime: Podman or Docker (auto-detected via `runner.DetectRuntime()`)

## Architecture

Interface-driven design. Core library in `internal/`, consumed by CLI (`cmd/rook/`) and GUI (`cmd/rook-gui/`).

- `internal/workspace/` — types (`Service`, `Workspace`, `Manifest`) and YAML parsing
- `internal/ports/` — `PortAllocator` interface, `FileAllocator` (JSON-backed), system port availability check
- `internal/registry/` — `Registry` interface, `FileRegistry` (JSON-backed)
- `internal/profile/` — `Resolve()` expands profiles into service lists
- `internal/envgen/` — Go template resolution for `{{.Host.x}}`/`{{.Port.x}}`, file template resolution
- `internal/health/` — `Check` type with HTTP, TCP, command variants; `ParseFromService` for structured healthcheck configs
- `internal/runner/` — `Runner` interface, `ProcessRunner`, `DockerRunner` with build support, `Reconnectable` interface for container adoption
- `internal/orchestrator/` — `TopoSort()` + `Orchestrator` (incremental start/stop, health check waiting, single-service start/stop/restart, container reconnection)
- `internal/discovery/` — `Discoverer` interface for compose (extracts build, command, env_file), mise, devcontainer
- `internal/api/` — `WorkspaceAPI` service layer for GUI (Wails bindings, event emitter, log buffer)
- `internal/cli/` — Cobra commands with shared `cliContext` for dependency initialization

## Commands

```bash
make build-cli          # Build CLI → bin/rook
make build-gui          # Build GUI → bin/rook-gui (requires npm, webkit2gtk)
make test               # Run all tests
make install            # Install both to $GOPATH/bin
go test ./internal/X/   # Test a single package
```

## CLI Usage

```bash
rook init <path>              # Register workspace (auto-discovers from docker-compose)
rook up [workspace] [profile] # Start services (foreground with log streaming)
rook up -d                    # Start detached
rook up --build               # Force rebuild of services with Dockerfiles
rook down [workspace]         # Stop all containers
rook restart [ws] [service]   # Restart service(s)
rook status [workspace]       # Show workspace/service status
rook logs [workspace] [svc]   # Tail container logs
rook ports                    # Show global port allocation table
rook ports --reset            # Clear allocations and stop containers
rook env <workspace>          # Show resolved environment variables
rook list                     # List registered workspaces
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
- Services can be containers (`image`), buildable containers (`build`), or processes (`command`)
- Port allocation: preferred ports tried first, system availability checked, then sequential from range 10000-60000
- Pinned ports (`pin_port`) error on conflict instead of reassigning
- Profile resolution: entries can be service names, group names, or `*` wildcard
- Template vars in environment: `{{.Host.x}}` resolves to container name (container-to-container) or `localhost` (processes), `{{.Port.x}}` resolves to internal port (containers) or allocated port (processes)
- Template vars in mounted config files (e.g., Caddyfile): resolved to `.rook/resolved/` temp copies before mounting
- Container networking: all workspace containers run on a shared `rook_<workspace>` network
- Container naming: `rook_<workspace>_<service>` — used for discovery, reconnection, status checks
- Container reconnection: `DockerRunner.Adopt` + `Orchestrator.Reconnect` re-discover running containers on CLI restart
- `env_file` support: passes `--env-file` to container runtime for loading project `.env` files
- Health checks integrated into startup: orchestrator waits for health before starting dependents
- Crash detection: 1-second pause after container start, checks status and shows last 20 log lines on crash

## What's Not Yet Implemented

- Process `env_file` support (loading .env into process services)
- `rook discover` command (re-scan and show changes)
- Auto-rebuild detection (prompt when Dockerfile changes)
- Automatic .env variable injection (update project .env with rook-allocated ports)
- `rook down --volumes` (remove container volumes on teardown)
- GUI visual manifest editor (Settings tab is a placeholder)
- GUI system tray (waiting for Wails v3)
- File watching / live reload
- `rookd` daemon for headless/remote management
