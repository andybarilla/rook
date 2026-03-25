# GUI Log Streaming

## Problem

The GUI has all the infrastructure for displaying logs (LogBuffer, BufferLog, LogViewer, `service:log` events) but nothing ever calls `BufferLog()`. When a service starts or crashes, logs are visible via `rook logs` CLI but not in the GUI.

## Approach

Orchestrator exposes `StreamServiceLogs`, WorkspaceAPI drives streaming goroutines that feed into `BufferLog()`.

## Changes

### runner/docker.go

Add `cmdReadCloser` wrapper implementing `io.ReadCloser`. Holds the underlying reader + `*exec.Cmd`. `Close()` closes the reader and calls `cmd.Wait()` for cleanup.

### orchestrator/orchestrator.go

Add `StreamServiceLogs(wsName, serviceName string) (io.ReadCloser, error)`:
- Looks up the handle from the internal handle map
- Type-asserts the runner (`*runner.DockerRunner` or `*runner.ProcessRunner`)
- Calls the appropriate `StreamLogs` method
- For Docker, wraps result with `cmdReadCloser`
- Returns `io.ReadCloser`

### api/workspace.go

Add `logCancels map[string]context.CancelFunc` field (keyed by `"ws/svc"`).

Add `startLogStream(wsName, serviceName string)`:
- Creates a cancellable context
- Calls `orch.StreamServiceLogs(wsName, serviceName)`
- Spawns goroutine: `bufio.Scanner` reads lines, calls `BufferLog(ws, svc, line)` for each
- Stores cancel func in `logCancels`
- On reader EOF or context cancel, cleans up

Add `stopLogStream(wsName, serviceName string)`:
- Calls cancel func, removes from map

Wire into lifecycle methods:
- `StartService`: call `startLogStream` after successful start
- `StartWorkspace`: call `startLogStream` for each service after `orch.Up`
- `StopService` / `StopWorkspace`: call `stopLogStream`
- New `ReconnectWorkspace(name string)`: calls `orch.Reconnect`, then `startLogStream` for each running service

## Data Flow

```
Runner.StreamLogs() -> io.ReadCloser
  -> Orchestrator.StreamServiceLogs() -> io.ReadCloser
    -> WorkspaceAPI goroutine reads lines
      -> BufferLog(ws, svc, line)
        -> logBuffer.Add() + emitter.Emit("service:log", ...)
          -> Frontend LogViewer receives event
```

## Lifecycle

- **Stream starts**: after StartService, StartWorkspace (per service), or ReconnectWorkspace (per running service)
- **Stream ends**: reader EOFs naturally (container stops / process exits), or cancel func called on Stop
- **Crash case**: container exits -> `docker logs -f` EOFs -> goroutine exits. Logs up to crash are buffered and visible.

## Testing

- `orchestrator_test.go`: test `StreamServiceLogs` returns a reader for a mock runner with streaming support
- `api/workspace_test.go`: test that `startLogStream` populates the log buffer with lines from a mock reader
