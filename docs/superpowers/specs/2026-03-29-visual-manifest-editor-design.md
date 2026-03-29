# Visual Manifest Editor Design

## Goal

Replace the ManifestEditor placeholder in the GUI with a working editor that lets users view and edit their workspace's `rook.yaml` manifest without leaving the app.

## Background

The manifest tab currently shows a placeholder message telling users to edit `rook.yaml` directly. The backend already has `SaveManifest` and `PreviewManifest` bindings but is missing a `GetManifest` binding to load the raw manifest for editing. The frontend has no manifest editing UI.

## Approach: Dual-Mode Editor

Two editing modes, toggled with a button:

1. **Form mode** (default) — structured editing of services, groups, profiles, and workspace metadata via form controls
2. **YAML mode** — raw YAML text editing with syntax highlighting for advanced users or fields not covered by the form

### Form Mode

Top-level sections, each collapsible:

**Workspace metadata**: name (read-only display), type (`single`/`multi` dropdown), root (text input)

**Services**: accordion list, one panel per service. Each panel shows:
- Type indicator (container/build/process) derived from filled fields
- Core fields: `image`, `command`, `path`, `working_dir`, `build`, `dockerfile`, `build_from`
- Ports: editable list of integers, `pin_port` toggle per port
- Environment: key-value editor with add/remove rows
- `depends_on`: multi-select from other service names
- `volumes`: editable string list
- `env_file`: text input
- `healthcheck`: text input (command string) or structured form (test/interval/timeout/retries)
- Add service / remove service buttons

**Groups**: key → multi-select of service names

**Profiles**: key → multi-select of service names and group names (plus `*` option)

### YAML Mode

A `<textarea>` (monospace) showing the full YAML. On switch from form→YAML, serialize current form state via `PreviewManifest`. On switch from YAML→form, parse the YAML and populate the form (show validation errors if parse fails).

### Save Flow

1. User edits in either mode
2. "Preview" button shows the YAML that will be written (via `PreviewManifest`)
3. "Save" button calls `SaveManifest`, shows toast on success/error
4. After save, emit `workspace:changed` so the sidebar and other tabs refresh

## API Changes

### New: `GetManifest(name string) (*workspace.Manifest, error)`

Reads and parses the workspace's `rook.yaml`, returning the full `Manifest` struct. This is needed because `GetWorkspace` returns a read-only `WorkspaceDetail` that omits many editable fields (volumes, healthcheck, build, env_file, pin_port, etc.).

### Change: `SaveManifest` should emit `workspace:changed`

Currently it writes the file silently. After saving, emit `workspace:changed` so the UI refreshes.

## Scope

### In scope
- `GetManifest` API binding
- Form-mode editor for all manifest fields
- YAML-mode raw editor
- Mode toggle with bidirectional serialization
- Save with preview and confirmation toast
- Unsaved changes warning when switching tabs/workspaces

### Out of scope
- Undo/redo
- Real-time collaboration
- Schema validation beyond YAML parse errors
- Drag-and-drop reordering
