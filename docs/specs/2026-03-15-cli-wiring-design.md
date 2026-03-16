# CLI Command Wiring — Design Spec

**Date:** 2026-03-15
**Status:** Draft

## Overview

Wire the 5 stub CLI commands (`up`, `down`, `restart`, `status`, `logs`) to the orchestrator, runners, and health checker. Adds a shared CLI context for dependency initialization, foreground log streaming with signal handling, and Docker-only mode for commands that run without a foreground process.

## Shared CLI Context

A new `internal/cli/context.go` provides shared initialization for commands that need the orchestrator:

```go
type cliContext struct {
    registry  *registry.FileRegistry
    portAlloc *ports.FileAllocator
    orch      *orchestrator.Orchestrator
    process   *runner.ProcessRunner
    docker    *runner.DockerRunner
}
```

A `newCLIContext()` constructor creates all dependencies from `$XDG_CONFIG_HOME/rook/`. A `loadWorkspace(name string)` helper resolves a workspace by name or by looking for `rook.yaml` in the current directory when no name is given.

## Commands

### `rook up [workspace] [profile]`

Starts services for a workspace profile and stays in the foreground streaming logs.

**Arguments:**
- `workspace` — workspace name (optional; defaults to current directory's `rook.yaml`)
- `profile` — profile name (optional; defaults to `"default"`, falls back to `"all"` if no `"default"` profile exists)

**Flags:**
- `-d`, `--detach` — start services and exit immediately. Docker containers persist; process services will die when the CLI exits.

**Foreground mode (default):**
1. Resolve workspace and profile
2. Allocate/confirm ports
3. Generate `.env` files
4. Start services in dependency order (topological sort)
5. After starting each service, wait for its health check (if defined) before starting dependents. Uses `health.WaitUntilHealthy` with the service's configured interval/timeout, or defaults (2s interval, 30s timeout). If a health check times out, stop the startup sequence, report which service failed and its last log output, and leave already-started services running.
6. Stream interleaved logs to stdout, color-coded by service name with `[service-name]` prefix
7. Block until Ctrl+C (SIGINT) or SIGTERM
8. On signal: print "Shutting down...", stop services in reverse dependency order, exit 0

**Detach mode (`-d`):**
1. Same as steps 1-5
2. Print summary of started services and their ports
3. Exit immediately

**Log streaming** uses goroutines reading from each runner's `Logs()` (for process services) or `docker logs -f` (for containers), multiplexing to stdout with per-service color codes. Colors cycle through: green, yellow, blue, purple, cyan, red — assigned by service index.

### `rook down [workspace]`

Stops all services in a workspace.

**Arguments:**
- `workspace` — workspace name (optional; defaults to current directory)

**Behavior:**
- If a foreground `rook up` process is running, it handles shutdown via signal. `rook down` is for standalone use.
- Finds rook-managed Docker containers by naming convention `rook_<workspace>_<service>` and stops/removes them.
- Process services cannot be tracked without a foreground process — `rook down` only manages containers.
- Reports what was stopped.

### `rook restart [workspace] [service]`

Restarts services.

**Arguments:**
- `workspace` — workspace name (optional; defaults to current directory)
- `service` — specific service name (optional; if omitted, restarts all running services)

**Behavior:**
- Docker-only mode: finds containers by naming convention, stops and re-starts them.
- If `service` is specified, only restart that one container.
- Reports what was restarted.

### `rook status [workspace]`

Shows workspace and service status.

**Arguments:**
- `workspace` — workspace name (optional)

**No workspace arg — show all workspaces:**
```
WORKSPACE     STATUS      SERVICES    PROFILE
skeetr        running     3/3         default
titlevision   partial     2/7         doc-pipeline
my-blog       stopped     0/2         -
```

**With workspace arg — show service detail:**
```
SERVICE     TYPE        STATUS      PORT
postgres    container   running     10000
redis       container   running     10001
app         process     unknown     10002
```

Status detection:
- Container services: check Docker via `docker inspect` using naming convention
- Process services: show `unknown` (cannot detect without foreground process)

Supports `--json` flag for structured output.

### `rook logs [workspace] [service]`

Tails logs from running containers.

**Arguments:**
- `workspace` — workspace name (optional; defaults to current directory)
- `service` — specific service name (optional; if omitted, interleave all)

**Behavior:**
- Docker-only: runs `docker logs -f` on matching containers
- If no service specified: interleave logs from all workspace containers, color-coded with `[service-name]` prefix
- If service specified: show only that container's logs
- Streams until Ctrl+C

## Health Check Integration

The orchestrator's `Up` method currently starts all services without waiting for health checks. This spec adds health check waiting between dependent service starts:

1. After calling `runner.Start()` for a service, check if it has a healthcheck defined
2. If yes, parse the healthcheck using `health.ParseFromService(svc.Healthcheck)` to get the `Check` and `Config`
3. Call `health.WaitUntilHealthy(ctx, check, config.Interval)` with a timeout derived from `config.Timeout`
4. If the health check passes, proceed to start the next service
5. If it times out, return an error identifying the failed service — the caller decides whether to stop or leave running services up

This change is in the orchestrator itself, not in the CLI layer. The CLI's `up` command benefits from it automatically.

## Docker Container Naming

Rook-managed containers follow the convention: `rook_<workspace>_<service>`

This is already used by `DockerRunner`. The `down`, `restart`, `status`, and `logs` commands use this convention to discover containers without needing the orchestrator's in-memory state.

A new helper `internal/runner/docker.go` function `FindContainers(prefix string) []string` lists containers matching the `rook_<workspace>_` prefix using `docker ps -a --filter name=<prefix> --format {{.Names}}`.

## Testing Strategy

- **CLI context:** Unit test `newCLIContext()` and `loadWorkspace()` with temp directories
- **Command logic:** Integration tests that create a workspace with a rook.yaml, run `rook up -d` (detach mode), verify `rook status` shows running containers, then `rook down` stops them. Requires Docker.
- **Health check integration:** Unit test in orchestrator with a mock health checker
- **Signal handling:** Manual test (Ctrl+C during `rook up`)
- **Docker discovery:** Unit test `FindContainers` with mock `docker ps` output
