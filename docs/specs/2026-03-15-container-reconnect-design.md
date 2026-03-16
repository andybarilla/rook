# Container Reconnection — Design Spec

**Date:** 2026-03-15
**Status:** Draft

## Overview

When the CLI or GUI exits, Docker containers keep running. Currently, restarting `rook up` destroys and recreates them (`docker rm -f`), losing non-volume state. The orchestrator also loses its in-memory handles, so it can't track what's already running.

This spec adds container adoption (reuse running containers instead of recreating) and orchestrator reconnection (re-populate handles from running Docker containers on startup).

## Changes

### 1. DockerRunner.Start — Adopt Running Containers

Modify `internal/runner/docker.go` `Start` method. Before removing and recreating, check if the container already exists:

- **Container exists and is running:** Adopt it — register in the `containers` map, return a handle. Do not remove or recreate.
- **Container exists but is stopped/exited:** Remove it, then create a new one (current behavior).
- **Container does not exist:** Create a new one (current behavior).

Check is done via `docker inspect -f {{.State.Status}} <name>`. If the command fails, the container doesn't exist.

### 2. Orchestrator.Reconnect — Re-populate Handles

Add a `Reconnect(ws workspace.Workspace) error` method to `internal/orchestrator/orchestrator.go`. It:

1. Uses `runner.FindContainers(fmt.Sprintf("rook_%s_", ws.Name))` to list containers matching the workspace prefix
2. For each container found, checks if it's running via `runner.ContainerStatus(name)`
3. If running, extracts the service name from the container name (strips the `rook_<workspace>_` prefix) and creates a `RunHandle{ID: serviceName, Type: "docker"}` in the handles map
4. Skips containers that aren't running

After `Reconnect`, the orchestrator's incremental switching logic will correctly see these services as `alreadyRunning` and skip them during `Up`.

### 3. CLI Context — Call Reconnect on Startup

Modify `internal/cli/context.go`. The `loadWorkspace` helper (or a new method) calls `orch.Reconnect(ws)` after creating the orchestrator. This ensures every CLI command that uses the orchestrator has an accurate picture of running containers.

The reconnect is cheap (one `docker ps` call + N `docker inspect` calls) and idempotent.

### 4. GUI API — Call Reconnect

The `WorkspaceAPI` methods that load a workspace (`StartWorkspace`, `StopWorkspace`, `GetWorkspace`, etc.) should call `Reconnect` before operating. This can be done in `loadWorkspace` or in the orchestrator creation path.

## What This Enables

| Scenario | Before | After |
|---|---|---|
| `rook up -d` → exit → `rook up` | Destroys and recreates all containers | No-op (already running) |
| `rook up -d` → exit → `rook status` | Shows running (Docker discovery) | Shows running (same, but orchestrator also knows) |
| `rook up -d` → exit → `rook down` | Stops containers (Docker discovery) | Stops containers (orchestrator-aware) |
| `rook up -d` → exit → `rook up` with different profile | Destroys all, starts new profile | Incremental switch (stops removed, starts new, keeps shared) |
| GUI relaunch | No awareness of running containers | Reconnects, shows correct state |

## Edge Cases

- **Container name collision from another tool:** `FindContainers` only matches `rook_<workspace>_` prefix. If another tool creates a container with this exact prefix, it would be adopted. This is unlikely given the naming convention.
- **Container running but unhealthy:** Adopted as-is. The health check only runs during initial startup, not during reconnection. This is intentional — reconnect is about awareness, not validation.
- **Process services:** Cannot be reconnected (they die with the CLI process). This is a known limitation documented in the CLI wiring spec. Reconnect only handles Docker containers.

## Testing Strategy

- **DockerRunner.Start adoption:** Unit test with a mock that simulates an already-running container. Integration test (requires Docker) that starts a container, creates a new DockerRunner, and verifies Start adopts it.
- **Orchestrator.Reconnect:** Unit test with mock runner that verifies handles are populated. Integration test that starts containers, creates a fresh orchestrator, calls Reconnect, and verifies Status returns the correct state.
- **CLI integration:** E2E test: `rook init`, `rook up -d`, exit, `rook up -d` again (should not recreate), `rook down`.
