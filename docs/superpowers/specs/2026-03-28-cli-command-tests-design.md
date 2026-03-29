# CLI Command Tests Design

## Goal

Add integration tests for `down`, `restart`, `logs`, `env`, `list`, `status`, and `ports` commands to ensure CLI commands work correctly end-to-end.

## Background

The project has two test patterns:
1. Unit tests in `internal/cli/` - test helper functions and isolated logic
2. E2E tests in `test/e2e/` - build binary and run full command execution

Currently, only `init` has E2E coverage. The roadmap tasks focus on integration tests rather than pure unit tests.

## Scope

### Commands to Test

| Command | Key Scenarios |
|---------|---------------|
| `down` | Stop containers, `-v` removes volumes, network cleanup, no containers case |
| `restart` | Single service, all services, unknown service error |
| `logs` | Container logs, process logs, multi-service mux, no services case |
| `env` | Print resolved env vars, template resolution |
| `list` | Empty list, registered workspaces, JSON output |
| `status` | All workspaces view, single workspace detail, service status |
| `ports` | Show allocations, `--reset` clears and stops containers |

### Test Categories

1. **Unit tests** (`internal/cli/`): Test helper functions like `processStatus`, `streamSingleContainer` logic, output formatting
2. **Integration tests** (`test/e2e/`): Build binary and test full command execution with real workspace setup

## Approach

Focus on integration tests in `test/e2e/` since:
- Commands depend heavily on external state (container runtime, filesystem)
- The E2E pattern already exists and works well
- Unit testing CLI commands with Cobra requires mocking many dependencies

Add unit tests only for:
- `processStatus()` helper in status.go
- Output formatting functions
- Pure logic that can be easily isolated

## Non-Goals

- Testing container runtime behavior (assumed to work)
- Testing commands that require a running daemon (future `rookd`)
- Performance or stress testing
