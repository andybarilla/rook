# Learnings

## 2026-03-03 — Task 002: Site Registry

### Surprises / gotchas
- `go test ./...` fails at the root package level due to `main.go` embedding `frontend/dist` which doesn't exist in the worktree (Wails build artifact). This is a pre-existing issue — internal package tests work fine with `go test ./internal/...`.
- `gh pr create` from a worktree doesn't auto-detect the remote branch — need `--head task/002-site-registry` flag explicitly.

### Pattern confirmations
- Existing `internal/config` package uses `_test` package suffix for external tests (e.g., `package config_test`) — followed the same convention for `registry_test`.
- TDD flow worked cleanly: write tests first, verify compilation failure, implement, verify pass.

### Tool / command tips
- `go test ./internal/registry/... -v` to run only the registry package tests.
- `go test ./internal/... -v` to run all internal package tests without hitting the Wails embed issue at root.
