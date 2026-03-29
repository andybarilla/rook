# CLI Command Tests Plan

## Prerequisites

- [ ] Create worktree for implementation

## Implementation Order

### Phase 1: Unit Tests for Helper Functions

1. **`status_test.go`** - Test `processStatus()` helper
   - PID file exists and process alive → running
   - PID file exists but process dead → stopped (cleans up stale PID)
   - No PID file → stopped

2. **`list_test.go`** - Test list command with mocked registry
   - Empty registry
   - Multiple workspaces

### Phase 2: E2E Tests for Simple Commands

3. **Add to `test/e2e/init_test.go`** or create separate files:
   - `list_test.go` - List registered workspaces
   - `ports_test.go` - Show allocations, reset flag
   - `env_test.go` - Print resolved env vars

### Phase 3: E2E Tests for Container-Dependent Commands

These require a running container runtime but don't need actual containers started:

4. **`down_test.go`**
   - No containers case (just prints message, cleans network)
   - Tests container discovery mock or skip if no runtime

5. **`status_test.go`** (e2e)
   - All workspaces view
   - Single workspace detail

### Phase 4: Full Integration Tests (Optional)

6. **`restart_test.go`** - Requires running containers
7. **`logs_test.go`** - Requires running containers with log output

These may need to be skipped in CI if no container runtime is available.

## Test Helpers to Add

```go
// test/e2e/helpers.go
func buildRook(t *testing.T) string    // Build binary, return path
func createTestWorkspace(t *testing.T, manifest string) (dir string, configDir string)
func skipIfNoRuntime(t *testing.T)     // Skip test if podman/docker not available
```

## Files to Create/Modify

### Create
- `internal/cli/status_test.go` - Unit tests for processStatus
- `test/e2e/list_test.go` - E2E for list command
- `test/e2e/ports_test.go` - E2E for ports command
- `test/e2e/env_test.go` - E2E for env command
- `test/e2e/down_test.go` - E2E for down command
- `test/e2e/helpers.go` - Shared test utilities

### Modify
- `test/e2e/init_test.go` - Extract helper functions to `helpers.go`

## CI Considerations

- Simple command tests (list, ports, env) run without container runtime
- Container-dependent tests (down, restart, logs, status) check for runtime and skip if unavailable
- GitHub Actions already has container runtime available for e2e tests

## Estimated Scope

- ~10-15 test functions across 5-6 files
- Focus on happy paths and key error cases
- Not exhaustive edge case coverage
