# Process Service Status Design

## Problem

`rook status` shows "unknown" for process services because:

1. The CLI status command bypasses the orchestrator and only queries Docker/Podman containers
2. `ProcessRunner` tracks PIDs in an in-memory map that is lost on CLI restart
3. No mechanism exists to discover running processes from previous sessions

## Solution

PID file tracking with cross-platform liveness checks. When a process service starts, write a PID file. On status queries, read the PID file and check if the process is still alive.

## PID File Format & Location

**Location**: `<ws.Root>/.rook/.cache/pids/<service>.pid` (per workspace, consistent with the existing `.rook/.cache/logs/` convention)

**Path helper**: `PIDDirPath(wsRoot string) string` returns `<wsRoot>/.rook/.cache/pids/`

**Format** (JSON):
```json
{
  "pid": 12345,
  "command": "make run-api",
  "started_at": "2026-03-21T10:30:00Z"
}
```

**Lifecycle**:
- Created by `ProcessRunner.Start()` after the process launches
- Deleted by `ProcessRunner.Stop()` when the process is stopped
- Cleaned up when stale (PID dead)

## Cross-Platform Liveness Check

Use `os.FindProcess(pid)` followed by `process.Signal(syscall.Signal(0))`:
- Linux/macOS: returns nil if alive, error if dead
- Windows: `Signal` is unsupported — defer Windows-specific implementation to a future `pidfile_windows.go` build-tagged file. For now, `IsProcessAlive` returns false on unsupported platforms (process shows "stopped"), which is a safe default. Windows support is out of scope for this iteration.

**Staleness guard**: If PID is dead (signal fails with "no such process"), delete the PID file and report `stopped`.

## Package Structure

### New file: `internal/runner/pidfile.go`

```go
type PIDInfo struct {
    PID       int       `json:"pid"`
    Command   string    `json:"command"`
    StartedAt time.Time `json:"started_at"`
}

func PIDDirPath(wsRoot string) string  // returns <wsRoot>/.rook/.cache/pids/
func WritePIDFile(dir, serviceName string, info PIDInfo) error
func ReadPIDFile(dir, serviceName string) (*PIDInfo, error)
func RemovePIDFile(dir, serviceName string) error
func IsProcessAlive(pid int) bool
func ListPIDFiles(dir string) ([]string, error)  // returns service names
```

`IsProcessAlive` uses signal 0 on Linux/macOS. Windows returns false (safe default, deferred to future work).

## ProcessRunner Changes

### New field

- `pidDir string` — path to `.rook/.cache/pids/` for the workspace, set via `SetPIDDir(dir string)` (mirrors existing `SetLogDir` pattern)

### Start()

After launching the process, write the PID file with PID, command, and current timestamp.

### Stop()

Two paths depending on whether the entry is reconnected:

**Normal entries** (started by us): Call `entry.cancel()`, wait on `<-entry.done`, then remove PID file.

**Reconnected entries** (`cmd == nil`): Send `SIGTERM` via `os.FindProcess(pid).Signal(syscall.SIGTERM)`, poll `IsProcessAlive()` with a timeout (5s), escalate to `Kill()` if still alive, then remove PID file and clean up the entry.

### New: processEntry changes

```go
type processEntry struct {
    cmd         *exec.Cmd
    cancel      context.CancelFunc
    output      *syncBuffer
    logFile     *os.File
    done        chan struct{}
    err         error
    reconnected bool  // true if adopted from PID file, not started by us
    pid         int   // stored for reconnected entries (cmd is nil)
}
```

### Status()

For reconnected entries (where we don't own `cmd`), check PID liveness directly via `IsProcessAlive()` instead of the `done` channel. If the PID is dead, clean up the PID file and entry.

### New: Reconnect(serviceName string) (RunHandle, error)

- Read PID file for the service (requires `pidDir` to be set via `SetPIDDir`)
- Check if PID is alive via `IsProcessAlive()`
- If alive: create a `processEntry` with `reconnected: true`, `pid` set, `cmd: nil`, store in entries map, return `RunHandle`
- If dead: remove stale PID file, return error

## Orchestrator Changes

### Reconnect()

Currently only reconnects containers. Extend to:
- Derive PID directory from workspace root: `PIDDirPath(ws.Root)`
- Call `SetPIDDir` on the process runner before reconnecting
- Scan PID files via `ListPIDFiles()`
- Call `ProcessRunner.Reconnect(serviceName)` for each
- Store resulting handles in the `handles` map

The `Reconnect(ws workspace.Workspace)` signature does not change — `ws.Root` provides the path needed to derive the PID directory.

## CLI Status Command Changes

The CLI status command reads PID files directly (via `ReadPIDFile` and `IsProcessAlive` from the `pidfile` package) without going through the orchestrator. This mirrors how container status already works — `showWorkspaceDetail` calls `runner.ContainerStatus()` directly, not through the runner interface.

### `showWorkspaceDetail()` (`internal/cli/status.go`)

For process services, use PID file liveness check instead of hardcoding "unknown":
- Derive PID dir from workspace root: `PIDDirPath(ws.Root)`
- Read PID file for the service via `ReadPIDFile(pidDir, serviceName)`
- If PID file exists and process alive: "running"
- If PID file exists and process dead: "stopped" (clean up PID file)
- If no PID file: "stopped"

### `showAllWorkspaces()`

Update aggregate counting to include process service status:
- Iterate all services in the manifest
- For container services: check via `runner.ContainerStatus()` (existing behavior)
- For process services: check via `ReadPIDFile` + `IsProcessAlive`
- Compute `running` count from both service types
- Derive workspace status from `running` vs `total`: all running = "running", none = "stopped", mixed = "partial"

This requires loading the workspace manifest (already done for service counting) and having access to `ws.Root` for PID file paths.

## Testing

### `pidfile_test.go`
- Write/read/remove PID file round-trip
- List PID files returns correct service names
- `IsProcessAlive` returns true for a running subprocess
- `IsProcessAlive` returns false after killing the subprocess
- Read nonexistent PID file returns error
- `PIDDirPath` returns correct path

### `process_test.go` (new tests)
- PID file created on `Start()`, contains correct PID and command
- PID file removed on `Stop()`
- `Reconnect()` succeeds for alive process, returns valid handle
- `Reconnect()` fails and cleans up for dead process
- `Status()` on reconnected entry checks PID liveness
- `Stop()` on reconnected entry sends signal and removes PID file

### `orchestrator_test.go` (new tests)
- `Reconnect()` picks up process services from PID files
- `Reconnect()` skips stale PID files (dead processes)

### `status_test.go` (new or extended)
- Process services show "running" when PID alive
- Process services show "stopped" when no PID file
- Mixed workspace (containers + processes) shows correct aggregate status
