# Rook — Local Development Workspace Manager

**Date:** 2026-03-15
**Status:** Draft

## Problem

Developers working on multiple projects simultaneously face three recurring pain points:

1. **Port conflicts** — devcontainers, docker-compose services, and local dev servers all bind host ports. Running two projects means manually deconflicting 8080, 5173, 5432, etc.
2. **Environment file juggling** — `.env` files with hardcoded ports and hostnames must be manually updated when switching between local, devcontainer, and production contexts, or when ports change to avoid conflicts.
3. **Partial service orchestration** — complex systems (e.g., a microservice platform with 20+ services) require running different subsets depending on what you're working on. There's no good way to define and switch between these subsets.

## Solution

Rook is a local development workspace manager — a desktop app + CLI that:

- **Registers workspaces** by scanning existing project config (docker-compose, devcontainer, mise)
- **Allocates ports centrally** so no two workspaces ever conflict
- **Generates environment files** from a manifest with resolved ports and hostnames
- **Defines profiles** — named subsets of services to run together
- **Orchestrates services** — starts infra, waits for health, then starts app services

## Tech Stack

- **Language:** Go 1.24+
- **Desktop:** Wails (native webview)
- **Frontend:** React, Tailwind CSS v4, shadcn/ui
- **CLI:** Cobra
- **Testing:** TDD, interface-driven design

## Core Concepts

### Workspace

A workspace is a project (or collection of related projects) registered with Rook. Each workspace has:

- **Identity** — name, root path
- **Type** — `single` (one app + infra) or `multi` (multiple services sharing infra)
- **Services** — things that run (apps, databases, workers)
- **Profiles** — named subsets of services
- **Port allocations** — assigned from a global pool, stable across restarts

A workspace is defined by a `rook.yaml` manifest file at the project root.

### Manifest (`rook.yaml`)

```yaml
name: skeetr
type: single

services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: skeetr
      POSTGRES_PASSWORD: skeetr
      POSTGRES_DB: skeetr
    healthcheck: pg_isready -U skeetr
    volumes:
      - pg-data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    volumes:
      - redis-data:/data

  app:
    command: air
    ports: [8080]
    depends_on: [postgres, redis]
    environment:
      DATABASE_URL: "postgres://skeetr:skeetr@{{.Host.postgres}}:{{.Port.postgres}}/skeetr"
      REDIS_URL: "redis://{{.Host.redis}}:{{.Port.redis}}"

  frontend:
    command: npm run dev -- --port {{.Port.frontend}}
    ports: [5173]
    working_dir: frontend  # relative to workspace root

groups:
  infra:
    - postgres
    - redis

profiles:
  default:
    - infra
    - app
    - frontend
  backend-only:
    - infra
    - app
```

Multi-service example (titlevision-style):

```yaml
name: titlevision
type: multi
root: ~/dev/titlevision-ai

services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: tv
      POSTGRES_PASSWORD: tv
    healthcheck: pg_isready -U tv

  redis:
    image: redis:7-alpine

  pubsub:
    image: google/cloud-sdk:latest
    command: gcloud beta emulators pubsub start --host-port=0.0.0.0:{{.Port.pubsub}}

  agent-svc:
    path: agent-svc
    command: ./mvnw quarkus:dev -Ddebug=false -Dquarkus.http.port={{.Port.agent-svc}}
    ports: [8080]
    depends_on: [postgres, redis, pubsub]

  document-svc:
    path: document-svc
    command: ./mvnw quarkus:dev -Ddebug=false -Dquarkus.http.port={{.Port.document-svc}}
    ports: [8081]
    depends_on: [postgres, pubsub]

  llm-extraction-svc:
    path: llm-extraction-svc
    command: ./mvnw quarkus:dev -Ddebug=false -Dquarkus.http.port={{.Port.llm-extraction-svc}}
    ports: [8082]
    depends_on: [postgres, pubsub]

  prompt-svc:
    path: prompt-svc
    command: ./mvnw quarkus:dev -Ddebug=false -Dquarkus.http.port={{.Port.prompt-svc}}
    ports: [8083]
    depends_on: [postgres, redis]

groups:
  infra:
    - postgres
    - redis
    - pubsub

profiles:
  default:
    - infra
    - agent-svc
  doc-pipeline:
    - infra
    - document-svc
    - llm-extraction-svc
    - prompt-svc
  all:
    - infra
    - "*"
```

### Manifest Field Reference

| Field | Applies To | Description |
|---|---|---|
| `name` | workspace | Workspace identifier |
| `type` | workspace | `single` or `multi` |
| `root` | workspace | Root path (optional, defaults to manifest directory) |
| `services` | workspace | Map of service definitions |
| `groups` | workspace | Named collections of services |
| `profiles` | workspace | Named subsets for `rook up` |
| `image` | service | Docker image — makes this a container service |
| `command` | service | Shell command — makes this a process service |
| `path` | service | Subdirectory for multi-service workspaces (relative to root) |
| `working_dir` | service | Working directory for process services (relative to root or path) |
| `ports` | service | List of ports the service needs allocated |
| `environment` | service | Environment variables (supports `{{.Port.x}}` and `{{.Host.x}}` templates) |
| `depends_on` | service | Services that must be healthy before this one starts |
| `healthcheck` | service | Health check command (see Health Checks below) |
| `volumes` | service | Named volumes (`name:/container/path`). Namespaced per workspace to avoid collisions (e.g., `skeetr_pg-data`) |
| `pin_port` | service | Pin a specific host port instead of using the allocator (e.g., `pin_port: 8080`) |

### Volumes

Named volumes in the manifest are namespaced per workspace to prevent collisions. A volume `pg-data` in workspace `skeetr` becomes Docker volume `rook_skeetr_pg-data`. This means two workspaces can both define a `pg-data` volume without conflict.

Rook creates Docker volumes automatically on first `rook up`. Volumes persist across `rook down` / `rook up` cycles. `rook down --volumes` removes them.

### Health Checks

Services can define health checks in three forms:

- **Command:** `healthcheck: pg_isready -U skeetr` — runs a shell command inside the container (container services) or on the host (process services). Exit 0 = healthy.
- **HTTP:** `healthcheck: http://localhost:{{.Port.app}}/health` — GET request, 2xx = healthy.
- **TCP:** `healthcheck: tcp://{{.Host.postgres}}:{{.Port.postgres}}` — connection succeeds = healthy.

Default timeout: 30s. Default interval: 2s. Configurable per service:

```yaml
healthcheck:
  test: pg_isready -U skeetr
  interval: 5s
  timeout: 60s
  retries: 10
```

### Port Allocator

A global port registry at `~/.config/rook/ports.json` ensures no conflicts across workspaces.

- Ports are allocated individually per service, not in fixed blocks — the allocator picks the next available port from the global range (default: 10000-60000, configurable in `~/.config/rook/config.yaml`)
- Ports persist across restarts — `skeetr.postgres` is always the same port once allocated
- **Pinned ports** (`pin_port` in manifest): reserved globally. If two workspaces try to pin the same port, the second `rook init` fails with an error naming both workspaces and the conflicting port. The user must resolve the conflict by changing one manifest.
- The allocator validates at registration time (`rook init`), not at startup — so you get errors early
- `rook ports` shows the full allocation table across all workspaces

### Environment Generation

When `rook up` runs, it generates `.env` files with resolved values:

- Template variables (`{{.Port.postgres}}`, `{{.Host.redis}}`) resolve to allocated ports and correct hostnames
- **Context detection:** Rook checks the `DEVCONTAINER` environment variable (set by devcontainer runtimes) to determine context. When `DEVCONTAINER=true`, hosts resolve to Docker service names (e.g., `postgres`) and ports resolve to internal container ports (e.g., `5432`). Otherwise, hosts resolve to `localhost` and ports resolve to allocated host ports.
- The generated `.env` is written to the project directory and gitignored
- The manifest (`rook.yaml`) is the committed source of truth — `.env` is a build artifact

### Profiles

Profiles define which services to start together:

- Every workspace has an implicit `all` profile
- Custom profiles reference services or groups by name
- `"*"` wildcard includes all services — groups and wildcards are expanded and deduplicated, so listing both `infra` and `"*"` is valid and idempotent
- Switching profiles is a single command: `rook up titlevision doc-pipeline`
- Profile switches are incremental: services already running that are in the new profile stay up; services not in the new profile are stopped; new services are started

### Service Orchestration

`rook up` starts services in dependency order:

1. Resolve profile to service list
2. Confirm/allocate ports from global registry
3. Generate `.env` files with resolved ports and hostnames
4. Start services in dependency order (topological sort on `depends_on` — circular dependencies are detected and reported as an error at this step)
5. For each service:
   - **Container services** (`image` defined): pull and run via Docker
   - **Process services** (`command` defined): run as a local process
6. Wait for health checks before starting dependent services
7. Report status

**Failure handling:**
- **Health check timeout:** If a service doesn't become healthy within its timeout, Rook stops the startup, reports which service failed and its last log output, and leaves already-started services running (so you can debug). It does not attempt to start services that depend on the failed one.
- **Service crash after startup:** Rook monitors running services. If a service exits unexpectedly, it's marked as `crashed` in status output. No automatic restart — the user decides whether to `rook restart` or investigate. The GUI shows a visual indicator and the last few log lines.
- **Dependency failure:** If service B depends on service A and A fails to start or crashes, B is not started (or is stopped if it was running during a profile switch). Status output shows the dependency chain that caused the skip.

## Auto-Discovery

Rather than requiring users to write `rook.yaml` from scratch, Rook scans a project directory and generates a manifest from existing config:

### Sources

| Source | What Rook Extracts |
|---|---|
| `docker-compose.yml` | Services, images, ports, volumes, environment, depends_on, healthchecks |
| `.devcontainer/devcontainer.json` | Forwarded ports, features (runtime versions), post-create commands |
| `.devcontainer/docker-compose.yml` | Same as docker-compose, scoped to dev environment |
| `mise.toml` / `.tool-versions` | Runtime versions (Go, Node, Python, Java) |
| `Makefile` | Common dev commands (dev, test, migrate targets) |
| `package.json` | Node scripts (dev, start, build), dependencies |
| `.env.example` | Environment variable shape and defaults |

### Discovery Flow

**Single project:**
1. `rook init ~/dev/andybarilla/skeetr-app`
2. Scans directory, finds docker-compose + devcontainer + mise config
3. Presents findings: "Found: postgres 16, redis 7, Go app (port 8080), Vite frontend (port 5173), Go 1.24 via mise"
4. Generates `rook.yaml` with discovered services
5. User reviews, edits if needed, done

**Multi-service system:**
1. `rook init ~/dev/titlevision-ai`
2. Scans subdirectories, finds shared docker-compose + individual service dirs with their own build configs
3. Discovers services, shared infra, per-service databases
4. Prompts: "Which services do you typically run together?" to define profiles
5. Generates manifest with profiles pre-configured

### Re-Discovery

`rook discover [workspace]` re-scans an already-registered workspace (by name, not path — unlike `rook init` which takes a path) and shows what changed since last init. Non-destructive — shows a diff of what it would update, user confirms.

## GUI

### Tech

- Wails desktop app (Go backend, React frontend in webview)
- System tray with status indicator
- Same core library as CLI — GUI is a frontend to the same API

### Views

**Dashboard:**
- List of registered workspaces with status (running / stopped / partial)
- Active profile and running service count per workspace
- One-click start/stop per workspace
- Quick profile switcher dropdown

**Workspace Detail:**
- Service list with status indicators, allocated ports, restart/stop buttons per service
- Log viewer — multiplexed (all services, color-coded) or per-service tabs, with search/filter
- Environment tab — view generated `.env` with resolved values
- Settings tab — edit manifest visually (add/remove services, define profiles, pin ports)

**Discovery Wizard:**
- "Add Workspace" button opens directory picker
- Runs auto-discovery, shows findings
- Profile creation assistant for multi-service workspaces
- Generates and saves manifest

### System Tray

- Green icon: services running
- Gray icon: idle
- Click: open main window
- Right-click menu: quick start/stop for recent workspaces

## CLI

Full parity with GUI:

```
rook init <path>              # Scan project, generate rook.yaml
rook discover <workspace>     # Re-scan, show changes
rook up [workspace] [profile] # Start services (default profile if omitted)
rook down [workspace]         # Stop all services in workspace (--volumes to remove data)
rook restart [workspace] [service]  # Restart a service
rook status                   # Show all workspaces, running services, ports
rook logs <workspace> [service]     # Tail logs (all or specific service)
rook list                     # List registered workspaces
rook ports                    # Show global port allocation table
rook env <workspace>          # Print generated environment variables
```

JSON output via `--json` flag for scripting.

## Architecture

### Package Structure

```
cmd/
  rook/              # CLI entry point (Cobra)
  rook-gui/          # Wails GUI entry point

internal/
  core/              # App lifecycle, wiring, dependency injection
  workspace/         # Workspace model, CRUD, manifest parsing
  registry/          # Global workspace registry (~/.config/rook/workspaces.json)
  ports/             # Port allocator, global port registry
  envgen/            # Environment file generation, template resolution
  discovery/         # Auto-discovery from docker-compose, devcontainer, mise, etc.
  orchestrator/      # Service lifecycle, dependency ordering, start/stop
  runner/            # Service runners (docker, process)
    docker/          # Docker container runner
    process/         # Local process runner
  profile/           # Profile resolution, incremental switching
  health/            # Health check implementations
  cli/               # CLI command definitions

frontend/            # React SPA
  src/
    pages/
      Dashboard.tsx
      WorkspaceDetail.tsx
      DiscoveryWizard.tsx
    components/
      ServiceList.tsx
      LogViewer.tsx
      ProfileSwitcher.tsx
      PortTable.tsx
```

### Key Interfaces

```go
// Runner starts and stops a service
type Runner interface {
    Start(ctx context.Context, svc Service, ports PortMap) (RunHandle, error)
    Stop(handle RunHandle) error
    Status(handle RunHandle) (ServiceStatus, error)
    Logs(handle RunHandle) (io.ReadCloser, error)
}

// Discoverer extracts workspace config from a project directory
type Discoverer interface {
    Name() string  // e.g., "docker-compose", "devcontainer", "mise"
    Detect(dir string) bool
    Discover(dir string) (*DiscoveryResult, error)
}

// PortAllocator manages global port assignments
type PortAllocator interface {
    Allocate(workspace string, service string, preferred int) (int, error)
    Release(workspace string, service string) error
    Get(workspace string, service string) (int, bool)
    All() PortRegistry
}

// Orchestrator manages service lifecycle for a workspace
type Orchestrator interface {
    Up(ctx context.Context, ws Workspace, profile string) error
    Down(ctx context.Context, ws Workspace) error
    Restart(ctx context.Context, ws Workspace, service string) error
    Status(ws Workspace) ([]ServiceStatus, error)
}
```

### GUI-to-Core Communication

The Wails `App` struct exposes methods to the React frontend via Wails bindings (auto-generated TypeScript). The GUI calls the same core library as the CLI — the `App` struct is a thin adapter:

```go
type App struct {
    orchestrator Orchestrator
    registry     Registry
    ports        PortAllocator
    discovery    []Discoverer
}

// Exposed to frontend via Wails bindings
func (a *App) ListWorkspaces() ([]WorkspaceSummary, error)
func (a *App) GetWorkspace(name string) (*WorkspaceDetail, error)
func (a *App) Up(workspace, profile string) error
func (a *App) Down(workspace string) error
func (a *App) Restart(workspace, service string) error
func (a *App) GetLogs(workspace, service string) ([]LogEntry, error)
func (a *App) GetPorts() (PortRegistry, error)
func (a *App) InitWorkspace(path string) (*DiscoveryResult, error)
func (a *App) SaveWorkspace(manifest Manifest) error
```

Real-time updates (service status changes, new log lines) are pushed to the frontend via Wails event emitter. The frontend subscribes to events like `service:status-changed` and `service:log` to update the UI reactively via React state/hooks.

### Design Principles

- **Interface-driven** — all external dependencies injected as interfaces for testability
- **CLI and GUI are peers** — both consume the same core library, neither is privileged
- **Read existing config, don't replace it** — Rook generates its own manifest from your existing files, never modifies them
- **Stable by default** — ports allocated once and persisted, no surprises between restarts
- **Incremental adoption** — start with `rook init`, get value immediately, customize over time

## Non-Goals (v1)

- Cloud deployment or staging environments — this is strictly local dev
- Runtime version management (installing Go, Node, etc.) — use mise directly
- SSL certificate management — out of scope for workspace orchestration
- Replacing docker-compose — Rook can use docker-compose under the hood, it's an orchestration layer above it
- Windows support in v1 — Linux and macOS first
