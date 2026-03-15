# Rook

Local development workspace manager that eliminates port conflicts, automates environment file generation, and orchestrates service subsets via profiles.

## The Problem

Running multiple projects simultaneously means:
- **Port conflicts** — two projects both want 5432, 8080, 6379
- **Environment juggling** — `.env` files with hardcoded ports that break when you switch contexts
- **Partial orchestration** — no way to run "just the backend" or "just infra" from a 20-service stack

## How Rook Solves It

Rook registers workspaces and allocates ports from a global pool. No two services across any workspace will ever collide. Environment templates resolve automatically.

```yaml
# rook.yaml
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

  app:
    command: air
    ports: [8080]
    depends_on: [postgres, redis]
    environment:
      DATABASE_URL: "postgres://skeetr:skeetr@{{.Host.postgres}}:{{.Port.postgres}}/skeetr"
      REDIS_URL: "redis://{{.Host.redis}}:{{.Port.redis}}"

groups:
  infra:
    - postgres
    - redis

profiles:
  default:
    - infra
    - app
  infra-only:
    - infra
```

## Install

```bash
go install github.com/andybarilla/rook/cmd/rook@latest
```

## Quick Start

```bash
# Register a workspace (auto-discovers from docker-compose.yml if no rook.yaml exists)
rook init ~/dev/my-project

# See what's registered
rook list

# Check port allocations across all workspaces
rook ports

# View resolved environment variables
rook env my-project
```

## CLI Reference

```
rook init <path>              Register a workspace (auto-discovers if no rook.yaml)
rook discover <workspace>     Re-scan workspace and show changes
rook up [workspace] [profile] Start services
rook down [workspace]         Stop all services in workspace
rook restart [workspace] [svc] Restart a service
rook status                   Show all workspaces and running services
rook list                     List registered workspaces
rook ports                    Show global port allocation table
rook logs <workspace> [svc]   Tail logs
rook env <workspace>          Print generated environment variables
```

Add `--json` to any command for structured output.

## Manifest Reference

### Services

Each service is either a **container** (has `image`) or a **process** (has `command`):

| Field | Type | Description |
|-------|------|-------------|
| `image` | string | Docker image (makes it a container service) |
| `command` | string | Shell command (makes it a process service) |
| `ports` | []int | Ports to allocate from the global pool |
| `pin_port` | int | Pin to an exact port (errors on conflict) |
| `environment` | map | Env vars, supports `{{.Host.x}}` and `{{.Port.x}}` templates |
| `depends_on` | []string | Services that must start first |
| `healthcheck` | string/object | Health check (command, `http://...`, or `tcp://...`) |
| `volumes` | []string | Docker volumes |
| `working_dir` | string | Working directory (relative to workspace root) |

### Healthchecks

```yaml
# Simple (auto-detected as command, HTTP, or TCP)
healthcheck: pg_isready -U app
healthcheck: http://localhost:8080/health
healthcheck: tcp://localhost:5432

# Structured
healthcheck:
  test: pg_isready -U app
  interval: 5s
  timeout: 60s
  retries: 10
```

### Groups and Profiles

Groups name sets of services. Profiles compose groups and services into runnable subsets:

```yaml
groups:
  infra: [postgres, redis]
  apps: [api, frontend]

profiles:
  default: [infra, apps]
  backend: [infra, api]
  all: ["*"]              # wildcard = all services
```

### Auto-Discovery

`rook init` scans for existing config when no `rook.yaml` exists:
- `docker-compose.yml` / `compose.yml` — extracts services, ports, depends_on
- `.devcontainer/devcontainer.json` — detects forwarded ports
- `mise.toml` / `.tool-versions` — detects runtime versions

## Architecture

```
cmd/rook/          CLI entry point (Cobra)
internal/
  workspace/       Manifest parsing, workspace/service types
  ports/           Global port allocator (file-backed JSON)
  registry/        Workspace registry (file-backed JSON)
  profile/         Profile resolution (groups, wildcards, dedup)
  envgen/          Environment template resolution, .env generation
  health/          Health checks (HTTP, TCP, command)
  runner/          Service runners (process, Docker)
  orchestrator/    Dependency ordering, lifecycle management
  discovery/       Auto-discovery (compose, devcontainer, mise)
  cli/             CLI command definitions
```

## Development

```bash
go test ./...           # Run all tests
go build ./cmd/rook     # Build the binary
go run ./cmd/rook       # Run without building
```

## License

MIT
