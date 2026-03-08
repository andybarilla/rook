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

## 2026-03-04 — Task 005: rook-ssl Plugin

### Surprises / gotchas
- Modifying `BuildConfig` signature (adding `CertProvider` parameter) required updating all existing test call sites — plan accounted for this correctly.
- Go's `crypto/x509` stdlib is sufficient for local CA + cert generation — no need for external mkcert dependency. ECDSA P-256 keys generate fast enough for tests.
- The `Write` tool requires reading the file first in worktrees, even when doing a full rewrite. Must `Read` before `Write` for existing files.

### Pattern confirmations
- Interface-based testability continues to scale well: `CertStore` for SSL mirrors `CaddyRunner`/`UpstreamResolver` pattern.
- `ServicePlugin` interface fits SSL well — it manages certs (a service concern), not request routing.
- Separating mock-based plugin tests from real cert generation tests (two test files) keeps test concerns clean.
- `map[string]any` Caddy config approach extends naturally for TLS blocks (`tls.certificates.load_files` + `tls_connection_policies`).

### Tool / command tips
- `--head` flag on `gh pr create` still required from worktrees.
- `go test ./internal/... -v` now runs 34 tests across five packages (caddy, config, plugin, registry, ssl).

## 2026-03-04 — Task 006: rook-php Plugin

### Surprises / gotchas
- No surprises — plan was exact, implementation matched 1:1. Simplest task so far: single package, no modifications to existing code.
- Deleting from a map during `range` iteration (in `Stop()`) works fine in Go — each key is visited at most once.

### Pattern confirmations
- `FPMRunner` interface follows the same testability pattern as `CaddyRunner`, `UpstreamResolver`, and `CertStore`.
- Dual interface implementation (`RuntimePlugin` + `ServicePlugin`) works cleanly — `Start()`/`Stop()` manage pools while `Handles()`/`UpstreamFor()` route requests.
- Non-fatal error handling pattern (log + continue) from plugin manager applied at the pool level too — one failed pool doesn't block others.
- External test package convention (`package php_test`) continues across all packages.

### Tool / command tips
- `--head` flag on `gh pr create` still required from worktrees.
- `go test ./internal/... -v` now runs 47 tests across six packages (caddy, config, php, plugin, registry, ssl).

## 2026-03-04 — Task 007: Core Wiring

### Surprises / gotchas
- Plan had an unused `plugin` import in the test file — Go's strict unused import rules caught it immediately. Minor fix (remove import).
- Root package (`app.go`, `stubs.go`, `main.go`) can't be tested with `go test` in worktrees due to missing `frontend/dist` embed. Core logic is fully testable via `internal/core` tests instead.

### Pattern confirmations
- Interface-based dependency injection pays off hugely at the wiring layer — `Core` accepts `CaddyRunner`, `FPMRunner`, `CertStore` and tests use simple stubs.
- Registry `OnChange` listener wired in `NewCore` triggers Caddy reload automatically — no manual reload calls needed in `AddSite`/`RemoveSite`.
- Separating `Core` (testable) from `App` (Wails binding, untestable without frontend) keeps the architecture clean.
- All existing components composed without any modifications — the interfaces designed in earlier tasks fit together perfectly.

### Tool / command tips
- `--head` flag on `gh pr create` still required from worktrees.
- `go test ./internal/... -v` now runs 54 tests across seven packages (caddy, config, core, php, plugin, registry, ssl).

## 2026-03-04 — Task 008: GUI Site List

### Surprises / gotchas
- Wails auto-generates JS bindings from Go methods, but only when running `wails dev` or `wails build` — neither works in worktrees (missing `frontend/dist`). Had to manually update `App.js` and `App.d.ts` to match the Go methods.
- `Greet` method was already removed in Core wiring (task 007) — no cleanup needed.
- Frontend can't be build-tested in worktrees without the full Wails environment. Verification is limited to Go internal tests; Svelte components need manual `wails dev` testing.

### Pattern confirmations
- Svelte 3 component structure is clean for CRUD UIs — props + callbacks pattern works well for SiteList and AddSiteForm.
- Wails JS binding format is simple and predictable: `window['go']['main']['App']['MethodName'](args)`.
- Separating Go backend (fully tested via `internal/core`) from frontend (manual testing) keeps CI reliable.

### Tool / command tips
- `--head` flag on `gh pr create` still required from worktrees.
- Wails JS bindings live at `frontend/wailsjs/go/main/App.js` — update manually when adding new Go methods in worktrees.
