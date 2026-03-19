# Design: File-Backed Process Logs

## Problem

`rook logs` only streams container logs via `docker logs -f`. Process services are invisible — the command prints "No running containers found" even when processes are running. Process logs only exist in-memory during `rook up` and are lost when the session ends.

## Solution

Tee process stdout/stderr to persistent log files so `rook logs` can tail them independently of `rook up`.

## Architecture

### Log file location

`.rook/.cache/logs/<service>.log` under the workspace root. Follows the existing `.rook/.cache/` convention used by `build-cache.json` and `resolved/`.

Path helper: `logDirPath(wsRoot string) string` returns `filepath.Join(wsRoot, ".rook", ".cache", "logs")` — used by both `up.go` and `logs.go` to stay in sync.

### ProcessRunner changes

`NewProcessRunner()` signature is unchanged. A `SetLogDir(dir string)` method sets the log directory after construction (before `Start()`). This avoids threading workspace root through `newCLIContext()` which doesn't know it yet.

When `logDir != ""` and `Start()` is called:

1. Create `logDir` if it doesn't exist
2. Open `<logDir>/<name>.log` with `O_CREATE|O_APPEND|O_WRONLY`
3. Write a session separator line: `--- rook up <timestamp> ---`
4. Set `cmd.Stdout` and `cmd.Stderr` to `io.MultiWriter(&output, logFile)` — tees to both the in-memory buffer (for `StreamLogs` during `up`) and the file (for persistence)
5. Store `*os.File` on `processEntry`; close it in the done goroutine (after `cmd.Wait()` returns) and in `Stop()`
6. **If `cmd.Start()` fails after the file is opened, close the file before returning the error**

Using `O_APPEND` with session separators instead of `O_TRUNC` preserves crash logs from prior sessions for debugging while keeping a readable format. Log files grow over time but are bounded by the workspace's lifetime.

When `logDir` is `""` (tests, or before `SetLogDir` is called), behavior is unchanged — output goes only to the in-memory buffer.

### Thread safety note

The existing `bytes.Buffer` used for process output has a pre-existing race: `cmd` writes to it from the exec pipe copier goroutine while `StreamLogs` reads under `entry.mu`. The `MultiWriter` change doesn't make this worse (same write path), but doesn't fix it either. Replace `bytes.Buffer` + `entry.mu` with a `syncBuffer` that wraps writes and reads with internal locking. This is a targeted fix for the existing race, not scope creep.

### logs.go changes

The `rook logs` command currently only handles containers. It will be extended to also handle process services. **Both paths call `cctx.loadWorkspace(wsName)` to get the manifest and `ws.Root`.**

**Multi-service mode (no service arg):**

1. Call `cctx.loadWorkspace(wsName)` to get manifest and root path
2. Find containers as today via `FindContainers(prefix)`
3. For each process service (`svc.IsProcess()`), check if `logDirPath(ws.Root)/<name>.log` exists
4. For each existing log file, create a tail reader (see below) and feed it to `logMux.addStream()`
5. Container and process log streams are multiplexed together

**Single-service mode (`rook logs ws svcname`):**

1. Call `cctx.loadWorkspace(wsName)` to get manifest and root path
2. Look up the service in the manifest
3. If service is a process, tail its log file to stdout (or error if no log file exists)
4. If service is a container, stream via `docker logs -f` as today
5. If service not found in manifest, fall back to container name lookup (backward compat)

**File tailing implementation:**

A `tailFile(path string, ctx context.Context) (io.ReadCloser, error)` function:
- Opens the file for reading
- Creates an `io.Pipe`
- Spawns a goroutine that reads existing content, then polls every 200ms for new data
- The goroutine exits when `ctx` is cancelled (from SIGINT handler calling `cancel()`), which closes the write end of the pipe
- The caller must call `pr.Close()` on the returned reader during shutdown
- Returns an `io.ReadCloser` compatible with `logMux.addStream()`

### Caller updates

| Location | Change |
|---|---|
| `cli/context.go` | No change to `NewProcessRunner()` call |
| `cli/up.go` | After resolving workspace, call `cctx.process.SetLogDir(logDirPath(ws.Root))` before orchestrator starts services |
| `cmd/rook-gui/main.go` | No change — GUI sets log dir per-workspace when starting |
| `runner/process_test.go` | No constructor change; tests that want file logging call `SetLogDir()` |

### What doesn't change

- `StreamLogs()` — still reads from the (now thread-safe) buffer during `up`
- `DockerRunner` — container logs handled by the runtime
- `logMux` — unchanged, both stream types feed into it
- `Runner` interface — no new methods required
- `NewProcessRunner()` signature — unchanged

## File inventory

| File | Action |
|---|---|
| `internal/runner/process.go` | Add `logDir` field, `SetLogDir()`, `syncBuffer`, file tee in `Start()`, close in done goroutine/`Stop()` |
| `internal/runner/process_test.go` | Add tests for file logging and truncation |
| `internal/cli/logs.go` | Call `loadWorkspace`, handle process services, add `tailFile()` |
| `internal/cli/up.go` | Call `SetLogDir()` after workspace resolution; add `logDirPath()` helper |
| `internal/cli/context.go` | No changes needed |
| `cmd/rook-gui/main.go` | No changes needed |

## Testing

- `TestProcessRunner_FileLogging` — start a process with `SetLogDir`, verify log file is created and contains output including session separator
- `TestProcessRunner_FileLogging_AppendsSessions` — start twice with same logDir, verify both sessions' output present with separators
- `TestTailFile` — write content to a temp file, create a tail reader, verify it returns existing content and follows new writes appended after creation
- `TestTailFile_ContextCancellation` — verify goroutine exits promptly when context is cancelled
- `TestSyncBuffer` — verify concurrent reads and writes don't race (run with `-race`)
