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
    orchestrator  *orchestrator.Orchestrator
    registry      *registry.FileRegistry
    portAlloc     *ports.FileAllocator
    processRunner *runner.ProcessRunner
    dockerRunner  *runner.DockerRunner
    discoverers   []discovery.Discoverer
}
```

**Workspace management:**
- `ListWorkspaces() []WorkspaceInfo` — all registered workspaces with status summary
- `GetWorkspace(name string) WorkspaceDetail` — full detail including services, profiles, ports
- `AddWorkspace(path string) (*DiscoveryResult, error)` — run discovery, register workspace
- `RemoveWorkspace(name string) error` — stops any running services first, then unregisters workspace and releases ports
- `SaveManifest(name string, manifest Manifest) error` — save edited manifest to `rook.yaml`

**Service orchestration:**
- `StartWorkspace(name, profile string) error` — start services for a profile
- `StopWorkspace(name string) error` — stop all services in workspace
- `StartService(workspace, service string) error` — start a single stopped service
- `RestartService(workspace, service string) error` — restart a single service (stop + start)
- `StopService(workspace, service string) error` — stop a single service

Note: These methods require adding `StartService`, `StopService`, and `Restart` methods to the orchestrator (not yet in the core library — it currently only has `Up`, `Down`, and `Status`). These will be implemented as part of the GUI build.

**Read-only queries:**
- `GetPorts() []PortEntry` — global port allocation table
- `GetEnv(workspace string) (map[string]map[string]string, error)` — resolved env vars per service. Reads allocated ports from the port allocator and resolves templates directly — does not require the workspace to be running. Returns an error if the workspace has never been initialized (no port allocations exist).
- `GetLogs(workspace, service string, lines int) ([]LogLine, error)` — recent log lines. Pass empty string for `service` to get interleaved logs from all services. Logs are buffered in-process memory only — after a GUI relaunch, historical logs are unavailable. For Docker containers, `GetLogs` falls back to `docker logs` to retrieve history. For process services, logs are lost on relaunch (documented limitation).
- `PreviewManifest(manifest Manifest) (string, error)` — render a manifest struct as YAML string (for the Discovery Wizard preview)

### Real-time Events

The backend uses Wails' `runtime.EventsEmit` to push updates to the frontend. The frontend subscribes via `runtime.EventsOn`.

**Event types:**

| Event | Payload | Trigger |
|---|---|---|
| `service:status` | `{workspace, service, status, port}` | Service status changes. `status` is one of: `starting`, `running`, `stopped`, `crashed`. The `starting` status is emitted when a service is launched but before its health check passes (or immediately if no health check is defined). Requires adding `StatusStarting` to the runner package. |
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

**Profile Switcher:** Selecting a new profile from the dropdown immediately calls `StartWorkspace(name, newProfile)`, which performs incremental switching — services in both old and new profiles stay running, services not in the new profile are stopped, new services are started. No confirmation dialog; the switch is instant and incremental. The dropdown shows the currently active profile (or "stopped" if no profile is running).

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

**Initialization:** On mount, the LogViewer calls `GetLogs(workspace, "", 500)` to fetch the last 500 lines of interleaved history. It then subscribes to `service:log` events for that workspace and appends new lines in real-time. Per-service tabs call `GetLogs(workspace, serviceName, 500)` on first activation and filter incoming `service:log` events by service name client-side.

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
5. Confirm — shows generated `rook.yaml` preview (rendered via `PreviewManifest` API call), "Add Workspace" button

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
- Window close quits the GUI but leaves services running as managed processes/containers. Services persist independently of the GUI window — they can be managed via the CLI or by reopening the GUI.
- Relaunching the binary re-shows the window and reconnects to running services (the orchestrator re-discovers running containers/processes by name).
- Menu bar: File (Add Workspace, Quit), View (Dashboard), Help (About)
- "Quit" in the menu also leaves services running. To stop services, use Stop All or `rook down` from CLI.

## Deferred Features

These are explicitly out of scope for this spec:

- **System tray** — the original system spec defines a tray icon (green/gray, right-click menu). This is intentionally deferred from v1 because Wails v2 lacks native tray support. Will be added when Wails v3 stabilizes or via a third-party library (e.g., getlantern/systray).
- **Light theme** — dark only in v1. Can be added later with CSS variables.
- **Drag-to-reorder** — sidebar workspace order, service order. Nice-to-have, not essential.
- **Notifications** — desktop notifications for service crashes. Deferred until tray support exists.
- **Remote daemon** — `rookd` daemon for headless management. The service layer API is designed to make this extraction straightforward, but the daemon itself is a separate project.

## Testing Strategy

- **Service layer:** Unit tests with mock orchestrator/registry/ports. Test each API method.
- **Frontend:** Component tests with React Testing Library for key interactions (service list actions, profile switching, log filtering).
- **Integration:** Wails `wails dev` for manual testing during development.
- **E2E:** Manual smoke test checklist (add workspace, start services, view logs, switch profiles, stop).
