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
- `internal/buildcache/` — stale build detection, Dockerfile hashing, `.dockerignore` parsing
- `internal/settings/` — `Settings` struct with JSON persistence (e.g., `AutoRebuild`)
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
rook down [workspace]         # Stop all containers (-v to remove volumes)
rook restart [ws] [service]   # Restart service(s)
rook status [workspace]       # Show workspace/service status
rook logs [workspace] [svc]   # Tail container logs
rook ports                    # Show global port allocation table
rook ports --reset            # Clear allocations and stop containers
rook env <workspace>          # Show resolved environment variables
rook check-builds [workspace] # Check which services need rebuilding
rook discover <path>          # Run auto-discovery on a path without registering
rook agentmd [path]           # Update rook section in CLAUDE.md/AGENTS.md
rook list                     # List registered workspaces
```

## Development Conventions

- **TDD**: Write tests first, verify they fail, then implement
- **Interface-driven**: Define interfaces, implement against them
- **One responsibility per file**: Each file has a clear purpose
- **No external test frameworks**: Use stdlib `testing` package only
- **Commit style**: `feat:`, `fix:`, `test:`, `chore:` prefixes
- **Test files**: `_test.go` in same package (black-box: `package x_test`)
- **Worktrees**: All implementation work must be done in a git worktree, regardless of size

## Key Patterns

- File-backed persistence: `ports.json` and `workspaces.json` in `$XDG_CONFIG_HOME/rook/`
- Services can be containers (`image`), buildable containers (`build`), or processes (`command`)
- Port allocation: always from range 10000-60000 (compose preferred ports are ignored); system availability checked
- Pinned ports (`pin_port`) error on conflict instead of reassigning
- Profile resolution: entries can be service names, group names, or `*` wildcard
- Template vars in environment: `{{.Host.x}}` resolves to container name (container-to-container) or `localhost` (processes), `{{.Port.x}}` resolves to internal port (containers) or allocated port (processes)
- Template vars in mounted config files (e.g., Caddyfile): resolved to `.rook/.cache/resolved/` temp copies before mounting
- Container networking: all workspace containers run on a shared `rook_<workspace>` network
- Container naming: `rook_<workspace>_<service>` — used for discovery, reconnection, status checks
- Container reconnection: `DockerRunner.Adopt` + `Orchestrator.Reconnect` re-discover running containers on CLI restart
- `env_file` support: containers get `--env-file` passed to runtime; process services get env_file parsed, shell-expanded, template-resolved, and merged into environment (inline wins on conflict)
- Resolved env file mount: writes `.rook/resolved/<service>.env` with resolved templates and mounts over the container's `.env` so Makefiles that `-include .env` get rook-resolved values
- Health checks integrated into startup: orchestrator waits for health before starting dependents
- Crash detection: 1-second pause after container start, checks status and shows last 20 log lines on crash
- Shell variable expansion: `${VAR:-default}` in compose environment and port values
- Default port inference: well-known images (postgres, redis, mysql, etc.) get default ports even without explicit port mappings
- Devcontainer compose priority: `.devcontainer/docker-compose.yml` is discovered before root compose
- Devcontainer depends_on merge: dependencies from root compose are merged into devcontainer services, including across name mismatches (e.g., `app` vs `api`)
- Devcontainer script copy: start scripts are copied to `.rook/` during init with a warning to review for devcontainer-specific code
- Dockerfile field: supports `dockerfile` in compose build object form (e.g., `.devcontainer/Dockerfile`)
- Multiple port mapping: all declared ports are mapped, not just the first
- Process log files: `ProcessRunner` tees stdout/stderr to `.rook/.cache/logs/<service>.log` via `io.MultiWriter`; `rook logs` tails these alongside container logs; session separators with `O_APPEND` preserve crash logs across restarts
- Auto-rebuild detection: `rook up` checks for stale builds (Dockerfile changes, context file changes, missing images) and prompts to rebuild; cache stored in `.rook/build-cache.json`
- Shared builds (`build_from`): when multiple services share the same build context and Dockerfile, discovery auto-sets `build_from` on duplicates; the runner reuses the source service's image tag without rebuilding

## What's Not Yet Implemented

- `rook status` for process services (currently shows "unknown" without a daemon)
- CLI command tests for `down`, `restart`, `logs`, `env`, `list`, `status`, `ports`
- GUI system tray (waiting for Wails v3)
- File watching / live reload
- `rookd` daemon for headless/remote management
