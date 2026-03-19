# GUI CLI Parity - Phase 2

## Summary

Add remaining CLI parity features to the GUI:
- Re-scan workspace for service changes with selective merge
- Workspace removal from sidebar
- Global settings UI (auto-rebuild toggle)

## Motivation

Phase 1 brought build management and port reset to the GUI. Several workflows still require the CLI:

- `rook discover` — Detect new/removed services since init
- `rook` doesn't have a direct GUI equivalent for workspace removal
- `autoRebuild` setting exists but has no UI toggle

Users should be able to manage these common workflows entirely from the GUI.

## Design

### 1. Re-scan Workflow

**Trigger points:**
- "Re-scan" button in WorkspaceDetail header (next to Start/Stop)
- Right-click on workspace in Sidebar → "Re-scan for changes"

**Flow:**
```
User triggers Re-scan
       ↓
Call API: DiscoverWorkspace(name)
       ↓
Any changes detected?
       ↓
   ┌──No──→ Show toast: "No changes detected"
   │
   Yes
       ↓
Show DiscoverDiffDialog:
  - Section: "New services" with checkboxes (all checked by default)
  - Section: "Removed services" with checkboxes (all checked by default)
  - Actions: "Apply Changes", "Cancel"
       ↓
User selects which to apply → "Apply Changes"
       ↓
Call API: ApplyDiscovery(name, selectedNew[], selectedRemoved[])
       ↓
Manifest updated, services refreshed, workspace:changed event emitted
```

**DiscoverDiffDialog layout:**
```
┌─────────────────────────────────────────┐
│ Re-scan Results                    [×]  │
├─────────────────────────────────────────┤
│ New services (2)                        │
│   ☑ api      image: node:20            │
│   ☑ worker   build: ./worker           │
│                                         │
│ Removed services (1)                    │
│   ☑ old-api  (no longer in compose)    │
│                                         │
│           [Cancel]  [Apply Changes]     │
└─────────────────────────────────────────┘
```

**Dialog behavior:**
- Checkbox list for each section (new services, removed services)
- All checkboxes checked by default
- ServiceDiff shows name and summary (image/build for new, reason for removed)
- "Apply Changes" disabled if no checkboxes selected
- On apply: close dialog, show brief "Changes applied" toast

### 2. Workspace Removal

**Trigger:** Right-click on workspace in Sidebar → "Remove workspace"

**Flow:**
```
User clicks "Remove workspace"
       ↓
Show ConfirmDialog:
  Title: "Remove workspace?"
  Message: "This will stop all services and unregister '<name>'.
            The project files will not be deleted."
  Actions: "Cancel", "Remove"
       ↓
User confirms → Call API: RemoveWorkspace(name)
       ↓
Navigate to Dashboard, sidebar refreshes
```

**Sidebar context menu:**
- Appears on right-click of workspace item
- Two options: "Re-scan for changes", "Remove workspace"
- Menu closes on click or click-outside

### 3. Global Settings UI

**Location:** Dashboard footer area

```
┌─────────────────────────────────────────┐
│ Dashboard                               │
│ ...                                     │
│                                         │
│ Reset Ports                             │
│                                         │
├─────────────────────────────────────────┤
│ ⚙️ Auto-rebuild on stale   [toggle]    │
└─────────────────────────────────────────┘
```

**Behavior:**
- Toggle switch bound to `autoRebuild` setting
- Default state: ON (matches existing settings package default)
- Changes persist immediately via `SaveSettings()`
- No confirmation dialog (instant toggle)
- Label shows setting purpose clearly

### 4. WorkspaceDetail Tab Rename

Rename the "Settings" tab to "Manifest" to avoid confusion with global settings.

```
Before: services | logs | environment | builds | settings
After:  services | logs | environment | builds | manifest
```

## API Changes

### New Types

```go
// DiscoverDiff represents detected changes between manifest and current discovery.
type DiscoverDiff struct {
    Source          string        `json:"source"`          // "docker-compose", "devcontainer", etc.
    NewServices     []ServiceDiff `json:"newServices"`     // Services found but not in manifest
    RemovedServices []ServiceDiff `json:"removedServices"` // Services in manifest but not found
    HasChanges      bool          `json:"hasChanges"`
}

// ServiceDiff describes a single service change.
type ServiceDiff struct {
    Name   string `json:"name"`
    Image  string `json:"image,omitempty"`
    Build  string `json:"build,omitempty"`
    Reason string `json:"reason,omitempty"` // e.g., "No longer in docker-compose"
}
```

### New Methods

```go
// DiscoverWorkspace re-runs discovery and returns the diff against the current manifest.
func (w *WorkspaceAPI) DiscoverWorkspace(name string) (*DiscoverDiff, error)

// ApplyDiscovery applies selected changes to the manifest.
func (w *WorkspaceAPI) ApplyDiscovery(name string, newServices []string, removedServices []string) error
```

**DiscoverWorkspace behavior:**
1. Load workspace entry from registry
2. Run all discoverers on workspace path
3. Load current manifest
4. Compare discovered services with manifest services
5. Build DiscoverDiff with new/removed lists
6. Return diff (empty lists if no changes)

**ApplyDiscovery behavior:**
1. Validate inputs: reject service names not in discovered set (for new) or not in manifest (for removed)
2. Load current manifest
3. Run discovery to get fresh service definitions
4. Add services from `newServices` list (copy from discovery result)
5. Remove services in `removedServices` list (containers NOT stopped — see edge case)
6. Write updated manifest to `rook.yaml`
7. Allocate ports for any new services with port mappings; on failure, return error and abort (manifest not written)
8. Emit `workspace:changed` event

**Existing API used:**
- `RemoveWorkspace(name string) error` — already exists, just needs GUI wiring

## File Structure

```
internal/api/
  types.go              # Add: DiscoverDiff, ServiceDiff
  workspace.go          # Add: DiscoverWorkspace, ApplyDiscovery
  workspace_test.go     # Add: tests for new methods

cmd/rook-gui/frontend/src/
  components/
    DiscoverDiffDialog.tsx   # New: selective merge dialog
    ContextMenu.tsx          # New: reusable right-click menu
    Toast.tsx                # New: simple toast notification
    Sidebar.tsx              # Modify: add context menu
  pages/
    Dashboard.tsx            # Modify: add settings footer
    WorkspaceDetail.tsx      # Modify: add re-scan button, rename settings→manifest
  hooks/
    useWails.ts              # Modify: add new API types
    useToast.ts              # New: toast context/hook
  App.tsx                    # Modify: add ToastProvider
```

## Edge Cases

1. **Re-scan with workspace running:** Allowed. New services won't start until user manually starts them. Removed services' containers are **not** automatically stopped — user must stop them manually. This matches CLI `rook discover` behavior (manifest-only change).

2. **Apply discovery with no selection:** "Apply Changes" button disabled.

3. **Discovery fails (e.g., docker-compose deleted or workspace path inaccessible):** Show error toast with message from API.

4. **Concurrent re-scans:** Second click while first in progress shows loading state, doesn't trigger second API call.

5. **Remove workspace that's currently selected:** Navigate to Dashboard after removal.

6. **Settings save fails:** Show error toast, toggle reverts to previous state.

7. **Port allocation failure during ApplyDiscovery:** API returns error, manifest is not modified. User sees error toast.

8. **Invalid service names in ApplyDiscovery request:** API validates that new service names exist in discovery result and removed names exist in current manifest. Returns error for invalid names.

9. **Modified services (port/env changes in compose):** Not detected. Re-scan only finds added/removed services, not configuration changes. This matches CLI behavior and is out of scope for this phase.

## Out of Scope

- Per-workspace settings (global only)
- Undo for applied discovery changes
- Viewing full discovered service details before applying
- Editing individual service fields during apply (just add/remove)
- Detecting modified services (port/env changes) — only add/remove detection
- Automatic rollback on partial failure (atomic update only)

## Test Scenarios

**API layer:**
- DiscoverWorkspace returns correct diff for new services
- DiscoverWorkspace returns correct diff for removed services
- DiscoverWorkspace returns empty diff when no changes
- ApplyDiscovery adds selected services to manifest
- ApplyDiscovery removes selected services from manifest
- ApplyDiscovery rejects invalid service names
- ApplyDiscovery handles port allocation failure (manifest unchanged)
- ApplyDiscovery allocates ports for new services with port mappings

**Frontend:**
- Re-scan button shows loading state
- DiscoverDiffDialog renders new and removed sections
- DiscoverDiffDialog disables Apply when nothing selected
- Context menu appears on right-click
- Remove workspace shows confirmation
- Settings toggle persists immediately
