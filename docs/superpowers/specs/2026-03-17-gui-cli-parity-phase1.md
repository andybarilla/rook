# GUI CLI Parity - Phase 1

## Summary

Bring the GUI to parity with CLI for build management and port reset functionality. This includes:
- Global settings storage with auto-rebuild preference
- Build detection and rebuild prompts
- Build status visualization (indicator + dedicated tab)
- Port reset functionality

## Motivation

The GUI currently lacks several CLI features:
- `rook up --build` flag for forcing rebuilds
- Automatic stale build detection and prompts
- `rook check-builds` for viewing which services need rebuilding
- `rook ports --reset` for clearing port allocations

Users who prefer the GUI should have access to these essential workflows without dropping to the CLI.

## Design

### 1. Global Settings Storage

**File location:** `$XDG_CONFIG_HOME/rook/settings.json`

**Schema:**
```json
{
  "autoRebuild": true
}
```

**Default values:**
- `autoRebuild`: `true` (matches CLI behavior of auto-rebuilding missing images)

**API methods:**
- `GetSettings() Settings` - returns current settings with defaults applied
- `SaveSettings(settings Settings) error` - persists settings to file

### 2. Build Detection API

**New types:**
```go
type BuildStatus struct {
    Name          string   `json:"name"`
    HasBuild      bool     `json:"hasBuild"`
    Status        string   `json:"status"` // "up_to_date", "needs_rebuild", "no_build_context"
    Reasons       []string `json:"reasons,omitempty"`
}

type BuildCheckResult struct {
    Services      []BuildStatus `json:"services"`
    HasStale      bool          `json:"hasStale"`
}
```

**API methods:**
- `CheckBuilds(workspace string) (*BuildCheckResult, error)` - checks all services in a workspace
- `StartWorkspace(name, profile string, forceBuild bool) error` - modified to accept rebuild flag

**CheckBuilds behavior:**
1. Load workspace manifest
2. Load build cache from `.rook/.cache/build-cache.json`
3. For each service with a build context:
   - Check if image exists
   - Compare Dockerfile hash
   - Compare context file hashes
4. Return status for each service

### 3. Build Status Visualization

**Service row indicator:**
- Show an orange "build" badge/icon next to services that need rebuilding
- Badge only appears when `status === "needs_rebuild"`
- Positioned between the status dot and service name

**Dedicated "Builds" tab:**
- New tab in WorkspaceDetail: `services | logs | environment | builds | settings`
- Lists all services with their build status
- Shows:
  - Service name
  - Status icon and text (✅ Up to date, ⚠️ Needs rebuild, ○ No build context)
  - Reasons for rebuild (e.g., "Dockerfile modified", "src/main.go modified")
- Services are sorted: needs_rebuild first, then up_to_date, then no_build_context

### 4. Rebuild Prompt Flow

**On "Start" button click:**

```
User clicks Start
       ↓
Call CheckBuilds(workspace)
       ↓
Any services need rebuild?
       ↓
   ┌──No──→ StartWorkspace(name, profile, false)
   │
   Yes
       ↓
   Is autoRebuild enabled?
       ↓
   ┌──Yes──→ StartWorkspace(name, profile, true)
   │
   No
       ↓
   Show RebuildDialog
       ↓
   User choice:
   - "Rebuild" → StartWorkspace(name, profile, true)
   - "Skip"    → StartWorkspace(name, profile, false)
   - "Cancel"  → Close dialog, do nothing
```

**RebuildDialog content:**
- Title: "Rebuild Required"
- Body: "N service(s) need rebuilding:" followed by list of service names and primary reason
- Actions: "Rebuild", "Skip", "Cancel"

### 5. Port Reset

**API method:**
- `ResetPorts() error` - stops all rook containers, removes ports.json

**ResetPorts behavior:**
1. Iterate all registered workspaces
2. For each workspace, find containers with prefix `rook_<workspace>_`
3. Stop each container
4. Delete `ports.json` file
5. Ports will be re-allocated on next `StartWorkspace`

**Frontend:**
- "Reset Ports" button in Dashboard, below the port allocations table
- Styled as a secondary/danger action
- Confirmation dialog: "This will stop all running containers and clear port allocations. Continue?"
- Actions: "Cancel", "Reset Ports"
- On confirm → call `ResetPorts()`, refresh dashboard

## File Structure

```
internal/
  api/
    workspace.go        # Modify: StartWorkspace signature, add CheckBuilds, ResetPorts
    settings.go         # New: GetSettings, SaveSettings
    types.go            # Add: BuildStatus, BuildCheckResult, Settings
  settings/
    settings.go         # New: FileSettings, Load, Save, defaults

cmd/rook-gui/frontend/src/
  components/
    RebuildDialog.tsx   # New: prompt dialog for rebuild decision
    BuildStatusBadge.tsx # New: indicator for service row
    ConfirmDialog.tsx   # New: reusable confirmation dialog
    ServiceList.tsx     # Modify: integrate BuildStatusBadge
  pages/
    WorkspaceDetail.tsx # Modify: add Builds tab
    Dashboard.tsx       # Modify: add Reset Ports button
  hooks/
    useWails.ts         # Modify: add new API types
    useSettings.ts      # New: settings context provider
  App.tsx               # Modify: wrap with SettingsProvider
```

## API Changes Summary

| Method | Type | Description |
|--------|------|-------------|
| `GetSettings()` | New | Returns current settings with defaults |
| `SaveSettings(settings)` | New | Persists settings to file |
| `CheckBuilds(workspace)` | New | Returns build status for all services |
| `StartWorkspace(name, profile, forceBuild)` | Modified | Added forceBuild parameter |
| `ResetPorts()` | New | Stops containers, clears port allocations |

## Edge Cases

1. **No build cache exists:** First run or cache deleted → all services with build context return `needs_rebuild` with reason "no build cache"

2. **Settings file missing:** Return defaults, create file on first `SaveSettings`

3. **ResetPorts while services running:** Stop containers first, then delete file. If stop fails, still delete file and log warning.

4. **CheckBuilds for workspace with no services:** Return empty `Services` array, `HasStale: false`

5. **StartWorkspace with forceBuild=true on service without build context:** Ignore for that service (only affects services with `svc.Build != ""`)

## Out of Scope

- Per-workspace settings (global only for now)
- Build log streaming (uses existing log infrastructure)
- Canceling in-progress builds
- Scheduled/automatic rebuild checks (only on explicit Start action)
