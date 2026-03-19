# Design: File-Backed Process Logs

## Problem

`rook logs` only streams container logs via `docker logs -f`. Process services are invisible — the command prints "No running containers found" even when processes are running. Process logs only exist in-memory during `rook up` and are lost when the session ends.

## Solution

Tee process stdout/stderr to persistent log files so `rook logs` can tail them independently of `rook up`.

## Architecture

### Log file location

`.rook/.cache/logs/<service>.log` under the workspace root. Follows the existing `.rook/.cache/` convention used by `build-cache.json` and `resolved/`.

### ProcessRunner changes

`NewProcessRunner(logDir string)` takes a log directory path. When `logDir != ""` and `Start()` is called:

1. Create `logDir` if it doesn't exist
2. Open `<logDir>/<name>.log` with `O_CREATE|O_TRUNC|O_WRONLY` (truncate on each start so logs don't grow unbounded)
3. Set `cmd.Stdout` and `cmd.Stderr` to `io.MultiWriter(&output, logFile)` — tees to both the in-memory buffer (for `StreamLogs` during `up`) and the file (for persistence)
4. Store `*os.File` on `processEntry`, close it in `Stop()` and in the done goroutine

When `logDir` is `""` (tests, or cases where workspace root isn't known), behavior is unchanged — output goes only to the in-memory buffer.

### logs.go changes

The `rook logs` command currently only handles containers. It will be extended to also handle process services:

**Multi-service mode (no service arg):**

1. Load workspace manifest to get service list
2. Find containers as today via `FindContainers(prefix)`
3. For each process service (`svc.IsProcess()`), check if `.rook/.cache/logs/<name>.log` exists
4. For each existing log file, create a tail reader (see below) and feed it to `logMux.addStream()`
5. Container and process log streams are multiplexed together

**Single-service mode (`rook logs ws svcname`):**

1. Load workspace manifest
2. If service is a process, tail its log file to stdout
3. If service is a container, stream via `docker logs -f` as today
4. If neither log file nor container exists, print an error

**File tailing implementation:**

A `tailFile(path string) (io.ReadCloser, error)` function:
- Opens the file
- Reads existing content (so you see recent logs, not just new ones)
- Polls every 200ms for new data (like `StreamLogs` does for the buffer)
- Returns an `io.ReadCloser` compatible with `logMux.addStream()`
- Stops when the pipe is closed (on SIGINT)

### Caller updates

| Location | Change |
|---|---|
| `cli/context.go` | `newCLIContext` doesn't know workspace root yet — pass `""` initially. The `up` command sets the log dir before starting services. |
| `cli/up.go` | After resolving workspace, call `cctx.process.SetLogDir(logDir)` before orchestrator starts services |
| `cmd/rook-gui/main.go` | Pass `""` — GUI can set it per-workspace when starting |
| `runner/process_test.go` | Pass `""` — no file logging in tests |

Alternative: add `SetLogDir(dir string)` method to `ProcessRunner` so it can be set after construction but before `Start()`. This avoids threading workspace root through `newCLIContext()` which doesn't know it yet.

### What doesn't change

- `StreamLogs()` — still reads from the in-memory buffer during `up`
- `DockerRunner` — container logs handled by the runtime
- `logMux` — unchanged, both stream types feed into it
- `Runner` interface — no new methods required

## File inventory

| File | Action |
|---|---|
| `internal/runner/process.go` | Add `logDir` field, `SetLogDir()`, file tee in `Start()`, close in `Stop()` |
| `internal/runner/process_test.go` | Update `NewProcessRunner("")`, add test for file logging |
| `internal/cli/logs.go` | Load manifest, handle process services, add `tailFile()` |
| `internal/cli/context.go` | Update `NewProcessRunner("")` call |
| `cmd/rook-gui/main.go` | Update `NewProcessRunner("")` call |

## Testing

- `TestProcessRunner_FileLogging` — start a process with logDir set, verify log file is created and contains output
- `TestProcessRunner_FileLogging_Truncates` — start twice, verify log file only contains second run's output
- `TestLogsCmd_ProcessService` — verify `rook logs` tails process log files
- `TestTailFile` — verify tail reader returns existing content and follows new writes
