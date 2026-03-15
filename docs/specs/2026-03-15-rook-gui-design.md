# Rook GUI — Design Spec

**Date:** 2026-03-15
**Status:** Draft

## Overview

A Wails v2 desktop application that provides a graphical interface to Rook's core library. The GUI is a frontend to the same packages the CLI uses — workspace management, port allocation, service orchestration, and auto-discovery.

System tray support is deferred until Wails v3 stabilizes or a third-party tray library is integrated.

## Architecture

### Separation of Concerns

The GUI introduces a **service layer** (`internal/api/`) that sits between the frontend and the core library. This layer:

- Wraps the orchestrator, registry, port allocator, and discovery packages
- Exposes methods via Wails bindings to the React frontend
- Emits real-time events for status changes and log streaming
- Serves as the future extraction point for a daemon (`rookd`) — the frontend would switch from Wails bindings to gRPC/HTTP without changing its own code

```
cmd/rook-gui/main.go              Wails entry point, window config
internal/api/
  workspace.go                    Service layer (Wails-bound methods)
  events.go                       Event emitter for real-time updates
frontend/
  src/
    App.tsx                        Root layout (sidebar + content area)
    pages/
      Dashboard.tsx                Stats overview + port table
      WorkspaceDetail.tsx          Tabbed view (Services, Logs, Env, Settings)
    components/
      Sidebar.tsx                  Workspace list with status indicators
      ServiceList.tsx              Service rows with status/port/actions
      LogViewer.tsx                Multiplexed + per-service tab logs
      ProfileSwitcher.tsx          Dropdown to switch active profile
      DiscoveryWizard.tsx          Add workspace flow (simple + guided)
      EnvViewer.tsx                Resolved environment variable display
      ManifestEditor.tsx           Visual manifest editor (Settings tab)
```

### Tech Stack

- **Desktop framework:** Wails v2
- **Backend:** Go (same binary, same core library as CLI)
- **Frontend:** React 19, TypeScript, Tailwind CSS v4, shadcn/ui
- **Build:** Wails CLI (`wails build`, `wails dev`)

### Service Layer API

The `WorkspaceAPI` struct is bound to the Wails app and exposes these methods to the frontend:

```go
type WorkspaceAPI struct {
    orchestrator *orchestrator.Orchestrator
    registry     *registry.FileRegistry
    portAlloc    *ports.FileAllocator
    processRunner *runner.ProcessRunner
    dockerRunner  *runner.DockerRunner
}
```

**Workspace management:**
- `ListWorkspaces() []WorkspaceInfo` — all registered workspaces with status summary
- `GetWorkspace(name string) WorkspaceDetail` — full detail including services, profiles, ports
- `AddWorkspace(path string) (*DiscoveryResult, error)` — run discovery, register workspace
- `RemoveWorkspace(name string) error` — unregister workspace, release ports
- `SaveManifest(name string, manifest Manifest) error` — save edited manifest to `rook.yaml`

**Service orchestration:**
- `StartWorkspace(name, profile string) error` — start services for a profile
- `StopWorkspace(name string) error` — stop all services in workspace
- `RestartService(workspace, service string) error` — restart a single service
- `StopService(workspace, service string) error` — stop a single service

**Read-only queries:**
- `GetPorts() []PortEntry` — global port allocation table
- `GetEnv(workspace string) (map[string]map[string]string, error)` — resolved env vars per service
- `GetLogs(workspace, service string, lines int) ([]LogLine, error)` — recent log lines

### Real-time Events

The backend uses Wails' `runtime.EventsEmit` to push updates to the frontend. The frontend subscribes via `runtime.EventsOn`.

**Event types:**

| Event | Payload | Trigger |
|---|---|---|
| `service:status` | `{workspace, service, status, port}` | Service starts, stops, crashes, becomes healthy |
| `service:log` | `{workspace, service, line, timestamp}` | New log line from a service |
| `workspace:changed` | `{workspace, action}` | Workspace added, removed, or manifest changed |

The frontend maintains React state from these events. No polling.

## Views

### Navigation

Persistent left sidebar with workspace list. Clicking a workspace navigates to its detail view. The sidebar is always visible. The "Dashboard" is the default view when no workspace is selected.

### Dashboard

Shown when no workspace is selected in the sidebar (initial state).

**Content:**
- Summary stats row: running service count, stopped service count, total ports allocated
- Port allocation table spanning all workspaces (workspace, service, port, pinned indicator)

**Interactions:**
- Clicking a workspace row in the sidebar navigates to its detail view

### Sidebar

- Section header: "Workspaces" (uppercase, muted)
- Each workspace shows:
  - Workspace name (bold)
  - Status indicator: green dot + "N/N running", amber dot + "N/M running", gray dot + "stopped"
  - Left border color matches status (green/amber/gray)
  - Selected workspace has highlighted background
- "Add Workspace" button at bottom (dashed border)

### Workspace Detail

Displayed when a workspace is selected in the sidebar. Header bar shows workspace name, path, profile switcher dropdown, and Stop All / Start All button.

**Four tabs:**

#### Services Tab

Service list with one row per service:
- Status dot (green = running, amber = starting/health-checking, red = crashed, gray = stopped)
- Service name (bold)
- Descriptor: Docker image name for containers, command for processes
- Allocated port (monospace)
- Action links: restart, stop (when running) or start (when stopped)

Services are listed in dependency order (same as startup order).

#### Logs Tab

Log viewer with tab bar:
- **"All" tab** — interleaved log lines from all running services, color-coded by service name with a `[service-name]` prefix
- **Per-service tabs** — one tab per service, showing only that service's log output
- Auto-scrolls to bottom. Scrolling up pauses auto-scroll; a "Jump to bottom" button appears.
- Search/filter input at the top filters visible log lines by text match
- Log lines show timestamp + content

#### Environment Tab

Read-only display of resolved environment variables per service:
- Grouped by service name
- Each variable shows the key, the template (from manifest), and the resolved value
- Useful for debugging port/host resolution

#### Settings Tab

Visual editor for the workspace manifest (`rook.yaml`):
- Add/remove services
- Edit service fields (image, command, ports, environment, depends_on, healthcheck, volumes)
- Manage groups (add/remove, assign services)
- Manage profiles (add/remove, compose from groups and services)
- Pin/unpin ports
- Save button writes back to `rook.yaml` on disk
- Validation feedback (e.g., circular dependency warning, unknown service reference)

This is a convenience editor — power users can still edit `rook.yaml` directly.

### Discovery Wizard

Triggered by "Add Workspace" in the sidebar. Opens as a modal dialog.

**Simple flow** (discovered services <= 3):
1. Directory picker
2. Single screen: shows discovered services with images/commands/ports
3. "Add Workspace" button — registers, allocates ports, done

**Guided flow** (discovered services >= 4):
1. Directory picker
2. Review discovered services — list with checkboxes, user can exclude services
3. Group services — drag or multi-select to create named groups (e.g., "infra", "apps")
4. Create profiles — compose profiles from groups and individual services, name them
5. Confirm — shows generated `rook.yaml` preview, "Add Workspace" button

The threshold (3 vs 4 services) is a UX heuristic, not a hard rule. The guided flow is always available via a "Customize" link on the simple screen.

## Visual Design

### Color Scheme (Dark Theme)

| Element | Color |
|---|---|
| Background (main) | `#1a1a2e` |
| Background (sidebar, cards) | `#16162a` |
| Background (inputs, inset) | `#1e1e3a` |
| Border | `#2a2a4a` |
| Text (primary) | `#e0e0f0` |
| Text (secondary) | `#a0a0c0` |
| Text (muted) | `#6b7280` |
| Running / healthy | `#4ade80` (green) |
| Partial / starting | `#f59e0b` (amber) |
| Crashed / error | `#ef4444` (red) |
| Stopped / idle | `#6b7280` (gray) |
| Accent (active tab, selection) | `#6366f1` (indigo) |
| Log: service colors | green, amber, blue (`#60a5fa`), purple (`#a78bfa`), cyan (`#22d3ee`), pink (`#f472b6`) |

### Typography

- System font stack (`system-ui, -apple-system, sans-serif`)
- Monospace for ports, log lines, environment values (`ui-monospace, monospace`)
- Section headers: 10px uppercase with letter-spacing

### Component Library

shadcn/ui components used:
- `Table` — port allocation table, environment display
- `Tabs` — workspace detail tabs, log service tabs
- `Select` — profile switcher
- `Button` — start/stop/restart actions
- `Dialog` — discovery wizard modal
- `Input` — search/filter, manifest editor fields
- `Badge` — status indicators
- `ScrollArea` — log viewer, service list

## Window Behavior

- Default size: 1200x800, resizable
- Minimum size: 900x600
- Sidebar width: 220px, fixed (not resizable in v1)
- Window close minimizes to background (services keep running)
- Reopening the window restores previous view state
- Menu bar: File (Add Workspace, Quit), View (Dashboard), Help (About)

## Deferred Features

These are explicitly out of scope for this spec:

- **System tray** — requires Wails v3 or third-party library. Will be added when stable tray support is available.
- **Light theme** — dark only in v1. Can be added later with CSS variables.
- **Drag-to-reorder** — sidebar workspace order, service order. Nice-to-have, not essential.
- **Notifications** — desktop notifications for service crashes. Deferred until tray support exists.
- **Remote daemon** — `rookd` daemon for headless management. The service layer API is designed to make this extraction straightforward, but the daemon itself is a separate project.

## Testing Strategy

- **Service layer:** Unit tests with mock orchestrator/registry/ports. Test each API method.
- **Frontend:** Component tests with React Testing Library for key interactions (service list actions, profile switching, log filtering).
- **Integration:** Wails `wails dev` for manual testing during development.
- **E2E:** Manual smoke test checklist (add workspace, start services, view logs, switch profiles, stop).
