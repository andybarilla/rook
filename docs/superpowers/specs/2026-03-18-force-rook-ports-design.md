# Design: Always Use Rook-Range Ports

**Date:** 2026-03-18
**Status:** Approved

## Problem

Currently, port allocation tries the compose-discovered preferred port first (e.g., postgres gets 5432), only falling back to the rook range (10000-60000) if that port is taken. This causes conflicts with host services running on standard ports. Since rook users are running multiple services, using standard ports is an anti-pattern — rook-range ports should be the default.

## Decision

Remove the preferred port parameter from `Allocate()` entirely. All non-pinned ports are allocated from the 10000-60000 range. Users who need a specific port can use `pin_port` in the manifest.

## Scope

### Changed

1. **`PortAllocator` interface** (`internal/ports/allocator.go`) — Remove `preferred int` parameter from `Allocate(workspace, service string) (int, error)`

2. **`FileAllocator.Allocate()`** (`internal/ports/allocator.go`) — Remove the "try preferred port first" branch. Logic becomes: check existing allocation, then scan range.

3. **Call sites** — Drop the `svc.Ports[0]` argument from all `Allocate()` calls:
   - `internal/cli/up.go` — port allocation loop
   - `internal/cli/init.go` — init-time allocation
   - `internal/orchestrator/orchestrator.go` — `Up()` and `StartService()`

4. **Tests** — Update allocator tests to match new signature. Tests verifying "preferred port is used" become "always allocates from range" tests.

### Unchanged

- `Service.Ports []int` — still needed for container internal port mappings (`-p` flag)
- `pin_port` / `AllocatePinned()` — explicit user decision, always respected
- Discovery — still extracts ports from compose (used for container port mappings)
- Environment templates — `{{.Port.x}}` still resolves to allocated port
- GUI / API layer — no changes needed

## Port Precedence

1. `pin_port` — exact port or error (unchanged)
2. Rook range allocation (10000-60000) — for everything else

## Migration

None. Existing allocations in `ports.json` are preserved. New allocations (after reset or new workspace) use rook range.
