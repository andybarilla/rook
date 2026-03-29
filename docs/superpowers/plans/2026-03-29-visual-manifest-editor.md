# Visual Manifest Editor Plan

## Prerequisites

- [ ] Create worktree for implementation

## Implementation Order

### Phase 1: Backend — `GetManifest` binding

1. **`internal/api/workspace.go`** — Add `GetManifest(name string) (*Manifest, error)` method
   - Look up workspace path from registry
   - `workspace.ParseManifest` on `rook.yaml`
   - Return the parsed manifest

2. **`internal/api/workspace.go`** — Update `SaveManifest` to emit `workspace:changed` event after successful write

3. **`internal/api/workspace_test.go`** — Tests for `GetManifest` (round-trip with `SaveManifest`)

4. **Regenerate Wails bindings** — `wails generate module` to create TS types for `GetManifest`

### Phase 2: Frontend — Form State Management

5. **`ManifestEditor.tsx`** — Core component rewrite
   - State: `manifest` (loaded from `GetManifest`), `dirty` flag, `mode` (form/yaml), `yamlText` (for YAML mode)
   - Load manifest on mount and when `workspaceName` changes
   - Track dirty state for unsaved changes warning

### Phase 3: Frontend — Form Mode

6. **Workspace metadata section** — name (read-only), type dropdown, root input

7. **Service editor** — Collapsible accordion per service
   - Text inputs for `image`, `command`, `path`, `working_dir`, `build`, `dockerfile`, `build_from`
   - Port list editor (add/remove integer inputs)
   - `pin_port` input
   - Key-value editor for `environment`
   - Multi-select for `depends_on` (from sibling service names)
   - String list editor for `volumes`
   - Text input for `env_file`
   - Healthcheck: string input or structured form toggle
   - Add/remove service controls

8. **Groups editor** — Key-value where values are multi-select of service names

9. **Profiles editor** — Key-value where values are multi-select of service names, group names, and `*`

### Phase 4: Frontend — YAML Mode

10. **YAML textarea** — Monospace textarea with the full YAML
    - Form→YAML: serialize via `PreviewManifest`
    - YAML→Form: parse YAML client-side, show errors if invalid

### Phase 5: Frontend — Save Flow

11. **Save/preview buttons** — Preview shows YAML diff/preview, Save calls `SaveManifest`
12. **Toast feedback** — Success/error toasts on save
13. **Dirty state guard** — Warn on tab/workspace switch with unsaved changes

### Phase 6: Polish

14. **Styling** — Match existing brutalist theme (rook-card, rook-border, rook-text classes)
15. **Edge cases** — Empty manifest, missing fields, large service lists
