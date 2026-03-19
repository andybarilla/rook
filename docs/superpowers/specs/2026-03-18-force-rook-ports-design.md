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
   - `internal/cli/up.go` — port allocation loop (guard `len(svc.Ports) > 0` stays — services without ports still skip allocation)
   - `internal/cli/init.go` — init-time allocation. Currently loops over `svc.Ports` calling `Allocate()` per port; replace with a single `Allocate()` call per service.
   - `internal/orchestrator/orchestrator.go` — two call sites: `Up()` and `StartService()`
   - `internal/api/workspace.go` — two call sites: `DiscoverAndInit()` and `ApplyDiscoveryDiff()`

4. **Tests** — Update to match new signature:
   - `internal/ports/allocator_test.go` — all `Allocate()` calls drop `preferred` arg; tests verifying "preferred port is used" become "always allocates from range" tests
   - `internal/api/workspace_test.go` — `stubPortAlloc.Allocate()` signature must drop `preferred` parameter

### Unchanged

- `Service.Ports []int` — still needed for container internal port mappings (`-p` flag)
- `pin_port` / `AllocatePinned()` — explicit user decision, always respected
- Discovery — still extracts ports from compose (used for container port mappings)
- Environment templates — `{{.Port.x}}` still resolves to allocated port

## Port Precedence

1. `pin_port` — exact port or error (unchanged)
2. Rook range allocation (10000-60000) — for everything else

## Migration

None. Existing allocations in `ports.json` are preserved. New allocations (after reset or new workspace) use rook range.
