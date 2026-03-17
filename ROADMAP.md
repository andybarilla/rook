# Rook Roadmap

## Vision

Rook eliminates the friction of running multiple development projects simultaneously. No port conflicts, no manual `.env` juggling, no "which services do I need?" confusion. One tool to register workspaces, allocate ports globally, and orchestrate services with dependency-aware startup.

---

## Completed

### Core Foundation
- [x] Port allocation from global pool (10000-60000) with conflict detection
- [x] Workspace registry with file-backed persistence
- [x] Manifest parsing (`rook.yaml` and docker-compose auto-discovery)
- [x] Profile resolution (groups, wildcards, service lists)
- [x] Environment template resolution (`{{.Host.x}}`, `{{.Port.x}}`)
- [x] Dependency-aware service ordering (topological sort)

### Container Runtime
- [x] Docker and Podman support (auto-detected, Podman preferred)
- [x] Container lifecycle (start, stop, restart, logs)
- [x] Health check integration (HTTP, TCP, command)
- [x] Container reconnection (adopt running containers on CLI restart)
- [x] Build support (Dockerfile detection, context awareness)
- [x] Auto-rebuild detection (prompts when Dockerfile or context files change)

### Process Services
- [x] Process runner for non-containerized services
- [x] `env_file` parsing, shell expansion, and template resolution for processes
- [x] Environment merging (inline env wins over env_file)

### CLI
- [x] `rook init` with auto-discovery from docker-compose, devcontainer, mise
- [x] `rook up/down/restart/status/logs` commands
- [x] `rook ports` with `--reset` for clearing allocations
- [x] `rook env` for inspecting resolved variables
- [x] `rook check-builds` for stale build detection
- [x] Foreground mode with log streaming
- [x] Detached mode (`-d`)
- [x] JSON output (`--json`)

### GUI
- [x] Wails v2 + React 19 + TypeScript + Tailwind CSS v4 + shadcn/ui
- [x] Workspace list and service status
- [x] Log viewing
- [x] Start/stop controls

### Developer Experience
- [x] Devcontainer compose priority (`.devcontainer/docker-compose.yml`)
- [x] Devcontainer `depends_on` merging
- [x] Start script copying to `.rook/`
- [x] Resolved env file mounting (for Makefiles that `-include .env`)
- [x] Crash detection with log output
- [x] Shell variable expansion in compose (`${VAR:-default}`)
- [x] Default port inference for well-known images (postgres, redis, mysql, etc.)
- [x] Multiple port mapping support
- [x] Build cache tracking (`.rook/build-cache.json`)

---

## In Progress

*No items currently in progress.*

---

## Planned

### High Priority

#### Idempotent `rook up`
Running `rook up` when services are already running should detect existing containers and skip them rather than attempting to start duplicates. Should report which services are already running and which (if any) need to be started.

#### Auto-scaffold on `rook init`
When initializing a new workspace, automatically:
- Add `.rook/` to `.gitignore`
- Generate a `CLAUDE.md` blurb for AI assistant context
- Offer to create a starter `rook.yaml` from discovered services

#### Force Rook Ports Flag
Add `--rook-ports` flag to `rook init` / `rook up` to ignore preferred ports from compose files and always allocate from the Rook range. Useful when you want complete control over port assignments.

### Medium Priority

#### File Watching / Live Reload
Watch for changes to:
- `rook.yaml` — reload workspace config
- Source files — trigger rebuilds for buildable services
- Mounted config files — re-resolve templates

Potential integration with tools like `air`, `watchexec`, or built-in fsnotify.

#### Shared Build / Image Alias
Allow multiple services to reference a single built image without rebuilding. Useful when several services share the same Dockerfile but have different configs.

```
services:
  api:
    build: .
    image_alias: myapp
    
  worker:
    image_alias: myapp  # Reuses api's build
    command: node worker.js
```

### Low Priority

#### `rookd` Daemon
Headless daemon for remote management:
- REST API for workspace operations
- WebSocket for log streaming
- Useful for shared dev servers or CI/CD integration

#### GUI Visual Manifest Editor
Replace the Settings tab placeholder with a visual editor for `rook.yaml`:
- Add/remove/edit services
- Configure ports, environment, dependencies
- Validate before saving

#### GUI System Tray
System tray integration for quick access to common operations. Blocked on Wails v3 which will have native tray support.

---

## Future Considerations

These are ideas that may or may not happen, depending on user feedback and use cases:

- **Multi-host orchestration** — Run services across multiple machines
- **Cloud provider integration** — Spin up cloud resources (RDS, ElastiCache) instead of local containers
- **Snapshot/restore** — Save and restore container state
- **Service templates** — Pre-configured service definitions (e.g., "add postgres with sensible defaults")
- **Workspace linking** — Share services across workspaces (e.g., one redis for all projects)

---

## Contributing

Found a bug or have a feature request? Open an issue on [GitHub](https://github.com/andybarilla/rook).

Want to contribute? Pull requests are welcome. Please follow the conventions in `CLAUDE.md`:
- TDD: write tests first
- Interface-driven design
- One responsibility per file
- Stdlib `testing` package only
- Conventional commits (`feat:`, `fix:`, `test:`, `chore:`)
