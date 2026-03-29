# Roadmap

## Workflow

1. Pick the next unstarted task
2. Create a spec in `docs/superpowers/specs/` and a plan in `docs/superpowers/plans/`
3. Implement in a worktree, open a PR
4. Mark the task as done

## Tasks

### CLI

- [x] **CLI command tests** — Integration tests for `down`, `restart`, `logs`, `env`, `list`, `status`, `ports` commands
- [x] **Process service status** — `rook status` reports actual state for process services via PID file tracking
- [ ] **Smart init** — Interactive multi-source discovery, local service helpers, LLM-assisted Dockerfile analysis

### GUI

- [x] **Visual manifest editor** — Replace the Settings tab placeholder with a real manifest editor
- [ ] **System tray** — Minimize to tray, show workspace status (blocked on Wails v3)

### Infrastructure

- [ ] **File watching / live reload** — Watch source files and auto-restart services on change
- [ ] **`rookd` daemon** — Headless/remote workspace management daemon
