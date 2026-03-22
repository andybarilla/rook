# Design: Process Crash Detection + PID Tracking

**Date:** 2026-03-22
**Issue:** #6 (process service improvements)

## Problem

Process services have no crash detection. When a process command exits immediately (bad syntax, missing dependency), the orchestrator doesn't notice — `rook up` reports success while the process is already dead. Container services get a 1-second crash check with last 20 log lines; processes get nothing.

Additionally, process services show "unknown" status in `rook status` because there's no persistent state tracking across CLI invocations.

## What exists (from `process-status` branch, rebased onto main)

- PID file management (`.rook/.cache/pids/<service>.pid`) with write/read/remove/list/liveness check
- `ProcessRunner.Status()` checks liveness via signal(0), distinguishes running/stopped/crashed
- `ProcessRunner.Logs()` returns in-memory output buffer as `io.ReadCloser` (already implements the `Runner` interface)
- `ProcessRunner.Reconnect()` adopts processes across CLI restarts via PID files
- Orchestrator reconnection extended for process services
- `rook status` shows process state via PID file liveness checks
- Comprehensive tests for PID tracking, reconnection, status, and logs

## What we're adding: Process crash detection

### Orchestrator crash check

Remove the `if svc.IsContainer()` gate on the existing 1-second crash check in `orchestrator.go` so it applies to both service types. After starting a process:

1. Wait 1 second (same as containers)
2. Call `Status()` to check if process is still alive
3. If crashed/stopped, read last 20 lines via `Logs()` (which reads the in-memory buffer — safe because `entry.done` is already closed when `Status()` returns crashed, meaning all output has been flushed)
4. Return error with log context — same behavior as containers

No new methods needed — `ProcessRunner` already implements `Logs()` on the `Runner` interface. The orchestrator's existing crash check code works unchanged for both service types once the gate is removed.

## Changes needed

1. **`internal/orchestrator/orchestrator.go`** — Remove the `if svc.IsContainer()` gate on the crash check (~line 132), making it apply to all service types. The existing code path using `r.Logs()` and `r.Status()` works for both runners.
2. Tests for the new behavior

## Scope boundaries

- Only adds crash detection (1-second check after start in `Up()`)
- `StartService()` does not get crash detection — known gap, same as containers today
- Does not add ongoing health monitoring or watchdog behavior
- No new runner methods or interface changes needed
- PID tracking and reconnection are already implemented in the branch

## Testing

- Orchestrator test: process that exits immediately is detected as crashed with log output
- Orchestrator test: process that stays running passes the crash check
- Orchestrator test: crashed process error includes last log lines
