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

## 2026-03-03 — Task 003: Plugin Interfaces + Host

### Surprises / gotchas
- No surprises — the plan was precise and implementation matched 1:1. Clean TDD cycle with zero adjustments needed.
- `mockRuntimePlugin` embedding `mockPlugin` works well for composing mock types that satisfy extended interfaces.

### Pattern confirmations
- `SiteSource` interface decoupling works cleanly — `registry.Registry` satisfies it without any adapter code since `List()` and `Get()` already exist.
- External test package convention (`package plugin_test`) continues to work well for testing exported API surface.
- Non-fatal plugin error handling (log + mark degraded) keeps the lifecycle simple — no need for complex recovery logic.

### Tool / command tips
- `--head` flag on `gh pr create` is still needed from worktrees.
- `go test ./internal/... -v` now runs 24 tests across three packages (config, plugin, registry).

## 2026-03-03 — Task 004: Caddy Manager

### Surprises / gotchas
- When branching from a different worktree (agent-1), the new worktree for the task branch must be created from `main` to get the registry and plugin packages that were merged there. The agent-1 branch only has docs/plans.
- The plan was exact — zero modifications needed between plan and implementation.

### Pattern confirmations
- Interface-based testability pattern (`CaddyRunner`, `UpstreamResolver`) matches the `Plugin`/`RuntimePlugin` pattern from task 003.
- `map[string]any` for Caddy JSON config generation works cleanly with `json.MarshalIndent`.
- External test package convention (`package caddy_test`) continues across all packages.

### Tool / command tips
- `--head` flag on `gh pr create` still required from worktrees.
- `go test ./internal/... -v` now runs 27 tests across four packages (caddy, config, plugin, registry).
