# Flock Roadmap

A cross-platform, open-source local development environment manager. Inspired by Laravel Herd, extensible via plugins to any language stack.

## Development Workflow

1. **Task Planning**

- Study the existing codebase and understand the current state
- Update `ROADMAP.md` to include the new task
- Priority tasks should be inserted after the last completed task
- **Finding the next task to plan:** Scan the roadmap top-down. Skip completed items (checked). Skip items that already have a design doc in `docs/plans/` — those are already planned. The first unchecked item with no design doc is the next thing to plan.

2. **Task Creation**

- Study the existing codebase and understand the current state
- Create a new task file in the `docs/tasks/` directory
- Name format: `XXX-description.md` (e.g., `001-scaffold.md`)
- Include high-level specifications, relevant files, acceptance criteria, and implementation steps
- Refer to last completed task in the `docs/tasks/` directory for examples. For example, if the current task is `012`, refer to `011` and `010` for examples.
- Note that these examples are completed tasks, so the content reflects the final state of completed tasks (checked boxes and summary of changes). For the new task, the document should contain empty boxes and no summary of changes. Refer to `000-sample.md` as the sample for initial state.

3. **Task Implementation**

- Follow the specifications in the task file
- Implement features and functionality
- Update step progress within the task file after each step
- Stop after completing each step and wait for further instructions

4. **Roadmap Updates**

- Mark completed tasks with ✅ in the roadmap
- Add reference to the task file (e.g., `See: docs/tasks/001-scaffold.md`)

## Development Phases

### Phase 1 — Foundation

Architecture design: `docs/plans/2026-03-03-flock-core-design.md`

- [x] Project scaffold (Go + Wails + CI) — See: docs/tasks/001-scaffold.md
- [x] Site registry — See: docs/tasks/002-site-registry.md
- [x] Plugin interfaces + host — See: docs/tasks/003-plugin-host.md
- [x] Caddy manager (embedded) — See: docs/tasks/004-caddy-manager.md
- [x] flock-ssl plugin (mkcert) — See: docs/tasks/005-flock-ssl.md
- [x] flock-php plugin (FPM pools) — See: docs/tasks/006-flock-php.md
- [x] Core wiring — See: docs/tasks/007-core-wiring.md
- [x] GUI: system tray + site list — See: docs/tasks/008-gui-site-list.md

### Phase 2 — Services

- [x] flock-databases plugin (MySQL, PostgreSQL, Redis) — See: docs/tasks/009-flock-databases.md

### Phase 3 — External Plugins

- [x] Plugin discovery and loading API — See: docs/tasks/010-plugin-discovery.md
- [x] flock-node plugin — See: docs/tasks/011-flock-node.md

### Phase 4 — UI/UX Polish

- [x] Integrate DaisyUI + Tailwind CSS (framework-agnostic component classes + dark theme) — See: docs/plans/2026-03-04-daisyui-integration-design.md
- [x] Error UX (dismissable banners, friendly messages, loading states) — See: docs/plans/2026-03-04-error-ux-design.md
- [x] Add Site form improvements (collapsible form, better layout, confirmation on remove) — See: docs/plans/2026-03-04-add-site-form-improvements-design.md
- [x] Site table: show Node Version column — See: docs/plans/2026-03-04-add-site-form-improvements-design.md
- [x] Empty state improvements (actionable guidance for sites and services) — See: docs/plans/2026-03-04-empty-state-improvements-design.md
- [ ] Accessibility (color contrast, keyboard shortcuts)

### Future — Potential

- [ ] flock-sdk (Go library for plugin authors — handles JSON-RPC boilerplate)
