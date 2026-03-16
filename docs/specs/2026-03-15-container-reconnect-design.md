# Container Reconnection — Design Spec

**Date:** 2026-03-15
**Status:** Draft

## Overview

When the CLI or GUI exits, Docker containers keep running. Currently, restarting `rook up` destroys and recreates them (`docker rm -f`), losing non-volume state. The orchestrator also loses its in-memory handles, so it can't track what's already running.

This spec adds container adoption (reuse running containers instead of recreating) and orchestrator reconnection (re-populate handles from running Docker containers on startup).

## Changes

### 1. DockerRunner — Adopt and Reconnect Methods

Modify `internal/runner/docker.go`:

**Modify `Start` — adopt running containers:**

Before removing and recreating, check if the container already exists:

- **Container exists and is running:** Adopt it — register in the `containers` map, return a handle. Do not remove or recreate.
- **Container exists but is stopped/exited:** Remove it, then create a new one (current behavior).
- **Container does not exist:** Create a new one (current behavior).

Check is done via `docker inspect -f {{.State.Status}} <name>`. If the command fails, the container doesn't exist.

**Add `Adopt(serviceName string)` method:**

Populates the `containers` map for a service without starting anything. Used by `Orchestrator.Reconnect` to register already-running containers so that subsequent `Stop`, `Status`, and `Logs` calls work correctly.

```go
func (r *DockerRunner) Adopt(serviceName string) RunHandle {
    containerName := r.containerName(serviceName)
    r.mu.Lock()
    r.containers[serviceName] = containerName
    r.mu.Unlock()
    return RunHandle{ID: serviceName, Type: "docker"}
}
```

**Add `Prefix() string` method:**

Returns the runner's prefix, so callers can derive the container search prefix without reconstructing it.

```go
func (r *DockerRunner) Prefix() string { return r.prefix }
```

### 2. Orchestrator.Reconnect — Re-populate Handles

Add a `Reconnect(ws workspace.Workspace) error` method to `internal/orchestrator/orchestrator.go`.

The orchestrator's `containerRunner` field is typed as `runner.Runner` (interface). To access `DockerRunner`-specific methods (`Adopt`, `Prefix`), `Reconnect` type-asserts to a new `Reconnectable` interface:

```go
// Reconnectable is implemented by runners that support discovering and adopting
// already-running services (e.g., DockerRunner).
type Reconnectable interface {
    Prefix() string
    Adopt(serviceName string) RunHandle
}
```

This interface lives in `internal/runner/runner.go`. `DockerRunner` satisfies it implicitly.

**Reconnect logic:**

1. Type-assert `o.containerRunner` to `Reconnectable`. If it doesn't implement the interface, return nil (process-only workspaces).
2. Call `runner.FindContainers(reconnectable.Prefix() + "_")` to list containers matching the workspace prefix.
3. Post-filter results with `strings.HasPrefix(name, prefix)` to avoid Docker's substring matching returning false positives.
4. For each matching container, check `runner.ContainerStatus(name)` — if running, extract the service name by stripping the prefix, call `reconnectable.Adopt(serviceName)`, and store the returned handle in `o.handles[ws.Name]`.
5. Skip containers that aren't running.

After `Reconnect`, the orchestrator's incremental switching logic correctly sees these services as `alreadyRunning` and skips them during `Up`.

Note: `runner.FindContainers` and `runner.ContainerStatus` are package-level functions (not on the `Runner` interface). The orchestrator calls them directly — this is acceptable since `Reconnect` is specifically about Docker container discovery, not a generic runner operation. The `Reconnectable` interface keeps the adoption path clean.

### 3. CLI Context — Call Reconnect on Startup

Modify `internal/cli/context.go`. Add a `reconnect(orch, ws)` call in the commands that create an orchestrator. Since the CLI creates a fresh orchestrator per command invocation (`newOrchestrator(wsName)`), the reconnect call goes right after orchestrator creation in the command implementations (`up.go`, `restart.go`), not in `loadWorkspace`.

The reconnect is cheap (one `docker ps` call + N `docker inspect` calls) and idempotent.

### 4. GUI API — Call Reconnect

The `WorkspaceAPI` currently holds a single shared `Orchestrator` constructed with one `DockerRunner`. For reconnection to work across workspaces, the GUI should call `Reconnect` per-workspace when loading workspace details.

Since the GUI's `Orchestrator` has a single `DockerRunner` with a fixed prefix, and workspaces have different prefixes, the GUI needs to create per-workspace `DockerRunner` instances for reconnection — similar to how the CLI does it in `newOrchestrator`. The simplest fix: the `WorkspaceAPI.loadWorkspace` method creates a temporary orchestrator with the correct prefix and calls `Reconnect`, then merges the discovered handles into the shared orchestrator's state.

Alternatively, the GUI can be refactored to use per-workspace orchestrators (like the CLI). This is a larger change and can be deferred — the GUI's `status` display already uses direct Docker inspection via the `GetWorkspace` method.

## What This Enables

| Scenario | Before | After |
|---|---|---|
| `rook up -d` → exit → `rook up` | Destroys and recreates all containers | No-op (already running) |
| `rook up -d` → exit → `rook status` | Shows running (Docker discovery) | Shows running (same, but orchestrator also knows) |
| `rook up -d` → exit → `rook down` | Stops containers (Docker discovery) | Stops containers (orchestrator-aware) |
| `rook up -d` → exit → `rook up` with different profile | Destroys all, starts new profile | Incremental switch (stops removed, starts new, keeps shared) |
| GUI relaunch | No awareness of running containers | Reconnects, shows correct state |

## Edge Cases

- **Container name collision (substring match):** Docker's `--filter name=` does substring matching, not prefix matching. `FindContainers("rook_myapp_")` could match a container named `other_rook_myapp_thing`. The implementation must post-filter results with `strings.HasPrefix(name, expectedPrefix)` to avoid false adoptions.
- **Container running but unhealthy:** Adopted as-is. The health check only runs during initial startup, not during reconnection. This is intentional — reconnect is about awareness, not validation.
- **Process services:** Cannot be reconnected (they die with the CLI process). This is a known limitation. Reconnect only handles Docker containers.
- **Port allocation consistency:** `ports.json` persists allocations across restarts. When reconnecting to a container, the port allocation in `ports.json` should already match what the container was started with. No port reconciliation is needed — if `ports.json` was manually deleted or modified, the allocated ports won't match the container's actual port bindings, but this is a user error scenario.

## Testing Strategy

- **DockerRunner.Start adoption:** Test that calling Start when a container already exists and is running returns a handle without recreating. Verify `r.containers` is populated.
- **DockerRunner.Adopt:** Test that Adopt populates `r.containers` and returns a valid handle.
- **Orchestrator.Reconnect:** Test with a mock `Reconnectable` that verifies handles are populated for running containers and skipped for stopped ones.
- **CLI integration:** E2E test (requires Docker): init workspace, `rook up -d`, verify containers running, create fresh orchestrator + reconnect, verify status shows running, `rook down`, verify containers stopped.
