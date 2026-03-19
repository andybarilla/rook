# GUI CLI Parity - Phase 2 Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development
> (if subagents available) or superpowers:executing-plans to implement this plan.

**Goal:** Add remaining CLI parity features: re-scan workspace for changes, workspace removal from sidebar, global settings UI, and rename settings tab to manifest.
**Architecture:** Extend API layer with DiscoverWorkspace and ApplyDiscovery methods, add context menu to sidebar, create DiscoverDiffDialog for selective merge, add settings toggle to Dashboard, rename tab in WorkspaceDetail.
**Tech Stack:** Go 1.22+, gopkg.in/yaml.v3, Wails v2, React 19, TypeScript, Tailwind CSS v4

---

## File Structure Map

### New Files
```
cmd/rook-gui/frontend/src/components/DiscoverDiffDialog.tsx   # Selective merge dialog
cmd/rook-gui/frontend/src/components/ContextMenu.tsx          # Reusable right-click menu
cmd/rook-gui/frontend/src/components/Toast.tsx                # Simple toast notification
cmd/rook-gui/frontend/src/hooks/useToast.ts                   # Toast context/hook
```

### Modified Files
```
internal/api/types.go               # Add DiscoverDiff, ServiceDiff types
internal/api/workspace.go           # Add DiscoverWorkspace, ApplyDiscovery methods
internal/api/workspace_test.go      # Add tests for new methods
cmd/rook-gui/frontend/src/hooks/useWails.ts       # Add new API types and methods
cmd/rook-gui/frontend/src/components/Sidebar.tsx  # Add context menu for workspace actions
cmd/rook-gui/frontend/src/pages/Dashboard.tsx     # Add settings footer with auto-rebuild toggle
cmd/rook-gui/frontend/src/pages/WorkspaceDetail.tsx  # Add re-scan button, rename settings→manifest
cmd/rook-gui/frontend/src/App.tsx                 # Add ToastProvider wrapper
```

---

## Task 1: API Types for Discovery Diff

**Files:**
- Modify: `internal/api/types.go`

- [ ] **Step 1: Add DiscoverDiff and ServiceDiff types**

Add to the end of `internal/api/types.go`:

```go
// DiscoverDiff represents detected changes between manifest and current discovery.
type DiscoverDiff struct {
	Source          string        `json:"source"`
	NewServices     []ServiceDiff `json:"newServices"`
	RemovedServices []ServiceDiff `json:"removedServices"`
	HasChanges      bool          `json:"hasChanges"`
}

// ServiceDiff describes a single service change.
type ServiceDiff struct {
	Name   string `json:"name"`
	Image  string `json:"image,omitempty"`
	Build  string `json:"build,omitempty"`
	Reason string `json:"reason,omitempty"`
}
```

- [ ] **Step 2: Verify build succeeds**
Run: `go build ./internal/api/`
Expected: No errors

- [ ] **Step 3: Commit**
Run: `git add internal/api/types.go && git commit -m "feat(api): add DiscoverDiff and ServiceDiff types"`

---

## Task 2: DiscoverWorkspace API Method

**Files:**
- Modify: `internal/api/workspace.go`
- Modify: `internal/api/workspace_test.go`

- [ ] **Step 1: Write the failing test for DiscoverWorkspace**

Add to `internal/api/workspace_test.go`:

```go
func TestDiscoverWorkspace_NoChanges(t *testing.T) {
	dir := t.TempDir()
	registryPath := filepath.Join(dir, "workspaces.json")
	settingsPath := filepath.Join(dir, "settings.json")

	reg, _ := registry.NewFileRegistry(registryPath)

	wsDir := filepath.Join(dir, "myproject")
	os.MkdirAll(wsDir, 0755)

	manifest := &workspace.Manifest{
		Name:     "myproject",
		Type:     workspace.TypeMulti,
		Services: map[string]workspace.Service{},
	}
	manifestData, _ := yaml.Marshal(manifest)
	os.WriteFile(filepath.Join(wsDir, "rook.yaml"), manifestData, 0644)

	reg.Register("myproject", wsDir)

	alloc := &stubPortAlloc{}
	orch := orchestrator.New(nil, nil, nil)
	a := api.NewWorkspaceAPIWithSettings(reg, alloc, orch, nil, settingsPath)

	diff, err := a.DiscoverWorkspace("myproject")
	if err != nil {
		t.Fatalf("DiscoverWorkspace failed: %v", err)
	}

	if diff.HasChanges {
		t.Error("expected no changes for empty workspace")
	}
}

func TestDiscoverWorkspace_DetectsNewServices(t *testing.T) {
	dir := t.TempDir()
	registryPath := filepath.Join(dir, "workspaces.json")
	settingsPath := filepath.Join(dir, "settings.json")

	reg, _ := registry.NewFileRegistry(registryPath)

	wsDir := filepath.Join(dir, "myproject")
	os.MkdirAll(wsDir, 0755)

	composeContent := `
services:
  web:
    image: nginx:latest
    ports:
      - "3000:80"
`
	os.WriteFile(filepath.Join(wsDir, "docker-compose.yml"), []byte(composeContent), 0644)

	manifest := &workspace.Manifest{
		Name:     "myproject",
		Type:     workspace.TypeMulti,
		Services: map[string]workspace.Service{},
	}
	manifestData, _ := yaml.Marshal(manifest)
	os.WriteFile(filepath.Join(wsDir, "rook.yaml"), manifestData, 0644)

	reg.Register("myproject", wsDir)

	alloc := &stubPortAlloc{}
	orch := orchestrator.New(nil, nil, nil)
	discoverers := []discovery.Discoverer{discovery.NewComposeDiscoverer()}
	a := api.NewWorkspaceAPIWithSettings(reg, alloc, orch, discoverers, settingsPath)

	diff, err := a.DiscoverWorkspace("myproject")
	if err != nil {
		t.Fatalf("DiscoverWorkspace failed: %v", err)
	}

	if !diff.HasChanges {
		t.Error("expected changes detected")
	}
	if len(diff.NewServices) != 1 {
		t.Errorf("expected 1 new service, got %d", len(diff.NewServices))
	}
	if diff.NewServices[0].Name != "web" {
		t.Errorf("expected new service 'web', got %s", diff.NewServices[0].Name)
	}
}

func TestDiscoverWorkspace_DetectsRemovedServices(t *testing.T) {
	dir := t.TempDir()
	registryPath := filepath.Join(dir, "workspaces.json")
	settingsPath := filepath.Join(dir, "settings.json")

	reg, _ := registry.NewFileRegistry(registryPath)

	wsDir := filepath.Join(dir, "myproject")
	os.MkdirAll(wsDir, 0755)

	composeContent := `
services:
  web:
    image: nginx:latest
`
	os.WriteFile(filepath.Join(wsDir, "docker-compose.yml"), []byte(composeContent), 0644)

	manifest := &workspace.Manifest{
		Name: "myproject",
		Type: workspace.TypeMulti,
		Services: map[string]workspace.Service{
			"old-service": {Image: "nginx:old"},
		},
	}
	manifestData, _ := yaml.Marshal(manifest)
	os.WriteFile(filepath.Join(wsDir, "rook.yaml"), manifestData, 0644)

	reg.Register("myproject", wsDir)

	alloc := &stubPortAlloc{}
	orch := orchestrator.New(nil, nil, nil)
	discoverers := []discovery.Discoverer{discovery.NewComposeDiscoverer()}
	a := api.NewWorkspaceAPIWithSettings(reg, alloc, orch, discoverers, settingsPath)

	diff, err := a.DiscoverWorkspace("myproject")
	if err != nil {
		t.Fatalf("DiscoverWorkspace failed: %v", err)
	}

	if !diff.HasChanges {
		t.Error("expected changes detected")
	}
	if len(diff.RemovedServices) != 1 {
		t.Errorf("expected 1 removed service, got %d", len(diff.RemovedServices))
	}
	if diff.RemovedServices[0].Name != "old-service" {
		t.Errorf("expected removed service 'old-service', got %s", diff.RemovedServices[0].Name)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**
Run: `go test ./internal/api/ -v -run TestDiscoverWorkspace`
Expected: FAIL — `DiscoverWorkspace` method doesn't exist

- [ ] **Step 3: Add DiscoverWorkspace method**

Add to `internal/api/workspace.go`:

```go
// DiscoverWorkspace re-runs discovery and returns the diff against the current manifest.
func (w *WorkspaceAPI) DiscoverWorkspace(name string) (*DiscoverDiff, error) {
	entry, err := w.registry.Get(name)
	if err != nil {
		return nil, err
	}

	manifestPath := filepath.Join(entry.Path, "rook.yaml")
	manifest, err := workspace.ParseManifest(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("loading manifest: %w", err)
	}

	result, err := discovery.RunAll(entry.Path, w.discoverers)
	if err != nil {
		return nil, fmt.Errorf("discovery failed: %w", err)
	}

	diff := &DiscoverDiff{
		Source:          result.Source,
		NewServices:     []ServiceDiff{},
		RemovedServices: []ServiceDiff{},
	}

	for name, svc := range result.Services {
		if _, exists := manifest.Services[name]; !exists {
			sd := ServiceDiff{Name: name}
			if svc.Image != "" {
				sd.Image = svc.Image
			}
			if svc.Build != "" {
				sd.Build = svc.Build
			}
			diff.NewServices = append(diff.NewServices, sd)
		}
	}

	for name := range manifest.Services {
		if _, exists := result.Services[name]; !exists {
			diff.RemovedServices = append(diff.RemovedServices, ServiceDiff{
				Name:   name,
				Reason: "No longer in discovery source",
			})
		}
	}

	sort.Slice(diff.NewServices, func(i, j int) bool { return diff.NewServices[i].Name < diff.NewServices[j].Name })
	sort.Slice(diff.RemovedServices, func(i, j int) bool { return diff.RemovedServices[i].Name < diff.RemovedServices[j].Name })

	diff.HasChanges = len(diff.NewServices) > 0 || len(diff.RemovedServices) > 0

	return diff, nil
}
```

- [ ] **Step 4: Run test to verify it passes**
Run: `go test ./internal/api/ -v -run TestDiscoverWorkspace`
Expected: PASS

- [ ] **Step 5: Commit**
Run: `git add internal/api/workspace.go internal/api/workspace_test.go && git commit -m "feat(api): add DiscoverWorkspace method for change detection"`

---

## Task 3: ApplyDiscovery API Method

**Files:**
- Modify: `internal/api/workspace.go`
- Modify: `internal/api/workspace_test.go`

- [ ] **Step 1: Write the failing test for ApplyDiscovery**

Add to `internal/api/workspace_test.go`:

```go
func TestApplyDiscovery_AddsServices(t *testing.T) {
	dir := t.TempDir()
	registryPath := filepath.Join(dir, "workspaces.json")
	portsPath := filepath.Join(dir, "ports.json")
	settingsPath := filepath.Join(dir, "settings.json")

	reg, _ := registry.NewFileRegistry(registryPath)

	wsDir := filepath.Join(dir, "myproject")
	os.MkdirAll(wsDir, 0755)

	composeContent := `
services:
  web:
    image: nginx:latest
    ports:
      - "3000:80"
  db:
    image: postgres:15
    ports:
      - "5432:5432"
`
	os.WriteFile(filepath.Join(wsDir, "docker-compose.yml"), []byte(composeContent), 0644)

	manifest := &workspace.Manifest{
		Name:     "myproject",
		Type:     workspace.TypeMulti,
		Services: map[string]workspace.Service{},
	}
	manifestData, _ := yaml.Marshal(manifest)
	os.WriteFile(filepath.Join(wsDir, "rook.yaml"), manifestData, 0644)

	reg.Register("myproject", wsDir)

	alloc, _ := ports.NewFileAllocator(portsPath, 10000, 60000)
	orch := orchestrator.New(nil, nil, alloc)
	discoverers := []discovery.Discoverer{discovery.NewComposeDiscoverer()}
	a := api.NewWorkspaceAPIFull(reg, alloc, orch, discoverers, settingsPath, portsPath)

	err := a.ApplyDiscovery("myproject", []string{"web"}, []string{})
	if err != nil {
		t.Fatalf("ApplyDiscovery failed: %v", err)
	}

	manifestPath := filepath.Join(wsDir, "rook.yaml")
	updated, err := workspace.ParseManifest(manifestPath)
	if err != nil {
		t.Fatalf("parsing updated manifest: %v", err)
	}

	if _, exists := updated.Services["web"]; !exists {
		t.Error("expected web service to be added to manifest")
	}
}

func TestApplyDiscovery_RemovesServices(t *testing.T) {
	dir := t.TempDir()
	registryPath := filepath.Join(dir, "workspaces.json")
	portsPath := filepath.Join(dir, "ports.json")
	settingsPath := filepath.Join(dir, "settings.json")

	reg, _ := registry.NewFileRegistry(registryPath)

	wsDir := filepath.Join(dir, "myproject")
	os.MkdirAll(wsDir, 0755)

	composeContent := `
services:
  web:
    image: nginx:latest
`
	os.WriteFile(filepath.Join(wsDir, "docker-compose.yml"), []byte(composeContent), 0644)

	manifest := &workspace.Manifest{
		Name: "myproject",
		Type: workspace.TypeMulti,
		Services: map[string]workspace.Service{
			"old-service": {Image: "nginx:old"},
		},
	}
	manifestData, _ := yaml.Marshal(manifest)
	os.WriteFile(filepath.Join(wsDir, "rook.yaml"), manifestData, 0644)

	reg.Register("myproject", wsDir)

	alloc, _ := ports.NewFileAllocator(portsPath, 10000, 60000)
	orch := orchestrator.New(nil, nil, alloc)
	discoverers := []discovery.Discoverer{discovery.NewComposeDiscoverer()}
	a := api.NewWorkspaceAPIFull(reg, alloc, orch, discoverers, settingsPath, portsPath)

	err := a.ApplyDiscovery("myproject", []string{}, []string{"old-service"})
	if err != nil {
		t.Fatalf("ApplyDiscovery failed: %v", err)
	}

	manifestPath := filepath.Join(wsDir, "rook.yaml")
	updated, err := workspace.ParseManifest(manifestPath)
	if err != nil {
		t.Fatalf("parsing updated manifest: %v", err)
	}

	if _, exists := updated.Services["old-service"]; exists {
		t.Error("expected old-service to be removed from manifest")
	}
}

func TestApplyDiscovery_RejectsInvalidServiceNames(t *testing.T) {
	dir := t.TempDir()
	registryPath := filepath.Join(dir, "workspaces.json")
	portsPath := filepath.Join(dir, "ports.json")
	settingsPath := filepath.Join(dir, "settings.json")

	reg, _ := registry.NewFileRegistry(registryPath)

	wsDir := filepath.Join(dir, "myproject")
	os.MkdirAll(wsDir, 0755)

	composeContent := `
services:
  web:
    image: nginx:latest
`
	os.WriteFile(filepath.Join(wsDir, "docker-compose.yml"), []byte(composeContent), 0644)

	manifest := &workspace.Manifest{
		Name:     "myproject",
		Type:     workspace.TypeMulti,
		Services: map[string]workspace.Service{},
	}
	manifestData, _ := yaml.Marshal(manifest)
	os.WriteFile(filepath.Join(wsDir, "rook.yaml"), manifestData, 0644)

	reg.Register("myproject", wsDir)

	alloc, _ := ports.NewFileAllocator(portsPath, 10000, 60000)
	orch := orchestrator.New(nil, nil, alloc)
	discoverers := []discovery.Discoverer{discovery.NewComposeDiscoverer()}
	a := api.NewWorkspaceAPIFull(reg, alloc, orch, discoverers, settingsPath, portsPath)

	err := a.ApplyDiscovery("myproject", []string{"nonexistent"}, []string{})
	if err == nil {
		t.Error("expected error for nonexistent service name")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**
Run: `go test ./internal/api/ -v -run TestApplyDiscovery`
Expected: FAIL — `ApplyDiscovery` method doesn't exist

- [ ] **Step 3: Add ApplyDiscovery method**

Add to `internal/api/workspace.go`:

```go
// ApplyDiscovery applies selected changes to the manifest.
func (w *WorkspaceAPI) ApplyDiscovery(name string, newServices []string, removedServices []string) error {
	entry, err := w.registry.Get(name)
	if err != nil {
		return err
	}

	manifestPath := filepath.Join(entry.Path, "rook.yaml")
	manifest, err := workspace.ParseManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("loading manifest: %w", err)
	}

	result, err := discovery.RunAll(entry.Path, w.discoverers)
	if err != nil {
		return fmt.Errorf("discovery failed: %w", err)
	}

	for _, svcName := range newServices {
		svc, exists := result.Services[svcName]
		if !exists {
			return fmt.Errorf("service %q not found in discovery result", svcName)
		}
		manifest.Services[svcName] = svc
	}

	for _, svcName := range removedServices {
		if _, exists := manifest.Services[svcName]; !exists {
			return fmt.Errorf("service %q not found in manifest", svcName)
		}
		delete(manifest.Services, svcName)
	}

	for _, svcName := range newServices {
		svc := result.Services[svcName]
		if len(svc.Ports) > 0 {
			if _, err := w.portAlloc.Allocate(name, svcName, svc.Ports[0]); err != nil {
				return fmt.Errorf("allocating port for %s: %w", svcName, err)
			}
		}
	}

	if err := workspace.WriteManifest(manifestPath, manifest); err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}

	w.emitter.Emit("workspace:changed", WorkspaceChangedEvent{Workspace: name})

	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**
Run: `go test ./internal/api/ -v -run TestApplyDiscovery`
Expected: PASS

- [ ] **Step 5: Commit**
Run: `git add internal/api/workspace.go internal/api/workspace_test.go && git commit -m "feat(api): add ApplyDiscovery method for selective manifest updates"`

---

## Task 4: Frontend - Add API Types to useWails

**Files:**
- Modify: `cmd/rook-gui/frontend/src/hooks/useWails.ts`

- [ ] **Step 1: Add new TypeScript types and API declarations**

Add to `cmd/rook-gui/frontend/src/hooks/useWails.ts` after the BuildCheckResult interface:

```typescript
export interface ServiceDiff {
  name: string
  image?: string
  build?: string
  reason?: string
}

export interface DiscoverDiff {
  source: string
  newServices: ServiceDiff[]
  removedServices: ServiceDiff[]
  hasChanges: boolean
}
```

Add to the WorkspaceAPI interface in the window.go declaration:

```typescript
DiscoverWorkspace(name: string): Promise<DiscoverDiff>
ApplyDiscovery(name: string, newServices: string[], removedServices: string[]): Promise<void>
```

- [ ] **Step 2: Verify TypeScript compiles**
Run: `cd cmd/rook-gui/frontend && npm run build`
Expected: No TypeScript errors

- [ ] **Step 3: Commit**
Run: `git add cmd/rook-gui/frontend/src/hooks/useWails.ts && git commit -m "feat(gui): add TypeScript types for discovery API"`

---

## Task 5: Frontend - Toast Component and Hook

**Files:**
- Create: `cmd/rook-gui/frontend/src/components/Toast.tsx`
- Create: `cmd/rook-gui/frontend/src/hooks/useToast.ts`

- [ ] **Step 1: Create the Toast component**

```tsx
// File: cmd/rook-gui/frontend/src/components/Toast.tsx
import { useEffect } from 'react'

interface ToastProps {
  message: string
  type?: 'success' | 'error' | 'info'
  onClose: () => void
}

export function Toast({ message, type = 'info', onClose }: ToastProps) {
  useEffect(() => {
    const timer = setTimeout(onClose, 3000)
    return () => clearTimeout(timer)
  }, [onClose])

  const bgColor = type === 'error' ? 'bg-red-600' : type === 'success' ? 'bg-green-600' : 'bg-rook-accent'

  return (
    <div className={`fixed bottom-4 right-4 ${bgColor} text-white px-4 py-2 rounded-md shadow-lg text-xs z-50`}>
      {message}
    </div>
  )
}
```

- [ ] **Step 2: Create the useToast hook**

```tsx
// File: cmd/rook-gui/frontend/src/hooks/useToast.ts
import { createContext, useContext, useState, useCallback, ReactNode } from 'react'
import { Toast } from '../components/Toast'

interface ToastState {
  message: string
  type: 'success' | 'error' | 'info'
}

interface ToastContextType {
  show: (message: string, type?: 'success' | 'error' | 'info') => void
}

const ToastContext = createContext<ToastContextType | null>(null)

interface ToastProviderProps {
  children: ReactNode
}

export function ToastProvider({ children }: ToastProviderProps) {
  const [toast, setToast] = useState<ToastState | null>(null)

  const show = useCallback((message: string, type: 'success' | 'error' | 'info' = 'info') => {
    setToast({ message, type })
  }, [])

  const hide = useCallback(() => {
    setToast(null)
  }, [])

  return (
    <ToastContext.Provider value={{ show }}>
      {children}
      {toast && <Toast message={toast.message} type={toast.type} onClose={hide} />}
    </ToastContext.Provider>
  )
}

export function useToast(): ToastContextType {
  const context = useContext(ToastContext)
  if (!context) {
    throw new Error('useToast must be used within a ToastProvider')
  }
  return context
}
```

- [ ] **Step 3: Commit**
Run: `git add cmd/rook-gui/frontend/src/components/Toast.tsx cmd/rook-gui/frontend/src/hooks/useToast.ts && git commit -m "feat(gui): add Toast component and useToast hook"`

---

## Task 6: Frontend - ContextMenu Component

**Files:**
- Create: `cmd/rook-gui/frontend/src/components/ContextMenu.tsx`

- [ ] **Step 1: Create the ContextMenu component**

```tsx
// File: cmd/rook-gui/frontend/src/components/ContextMenu.tsx
import { useEffect, useRef, ReactNode } from 'react'

interface ContextMenuItem {
  label: string
  onClick: () => void
  danger?: boolean
}

interface ContextMenuProps {
  x: number
  y: number
  items: ContextMenuItem[]
  onClose: () => void
}

export function ContextMenu({ x, y, items, onClose }: ContextMenuProps) {
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const handleClick = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        onClose()
      }
    }
    document.addEventListener('click', handleClick)
    return () => document.removeEventListener('click', handleClick)
  }, [onClose])

  return (
    <div
      ref={ref}
      className="fixed bg-rook-card border border-rook-border rounded-md shadow-lg py-1 z-50"
      style={{ left: x, top: y }}
    >
      {items.map((item, i) => (
        <button
          key={i}
          onClick={() => {
            item.onClick()
            onClose()
          }}
          className={`block w-full text-left px-3 py-1.5 text-xs ${
            item.danger ? 'text-rook-crashed hover:bg-red-500/10' : 'text-rook-text hover:bg-rook-border/50'
          }`}
        >
          {item.label}
        </button>
      ))}
    </div>
  )
}
```

- [ ] **Step 2: Commit**
Run: `git add cmd/rook-gui/frontend/src/components/ContextMenu.tsx && git commit -m "feat(gui): add ContextMenu component for right-click actions"`

---

## Task 7: Frontend - DiscoverDiffDialog Component

**Files:**
- Create: `cmd/rook-gui/frontend/src/components/DiscoverDiffDialog.tsx`

- [ ] **Step 1: Create the DiscoverDiffDialog component**

```tsx
// File: cmd/rook-gui/frontend/src/components/DiscoverDiffDialog.tsx
import { useState } from 'react'
import type { DiscoverDiff, ServiceDiff } from '../hooks/useWails'

interface DiscoverDiffDialogProps {
  open: boolean
  diff: DiscoverDiff | null
  onApply: (newServices: string[], removedServices: string[]) => void
  onCancel: () => void
}

export function DiscoverDiffDialog({ open, diff, onApply, onCancel }: DiscoverDiffDialogProps) {
  const [selectedNew, setSelectedNew] = useState<Set<string>>(new Set())
  const [selectedRemoved, setSelectedRemoved] = useState<Set<string>>(new Set())

  if (!open || !diff) return null

  const toggleNew = (name: string) => {
    const next = new Set(selectedNew)
    if (next.has(name)) {
      next.delete(name)
    } else {
      next.add(name)
    }
    setSelectedNew(next)
  }

  const toggleRemoved = (name: string) => {
    const next = new Set(selectedRemoved)
    if (next.has(name)) {
      next.delete(name)
    } else {
      next.add(name)
    }
    setSelectedRemoved(next)
  }

  const selectAllNew = () => {
    setSelectedNew(new Set(diff.newServices.map(s => s.name)))
  }

  const selectAllRemoved = () => {
    setSelectedRemoved(new Set(diff.removedServices.map(s => s.name)))
  }

  const canApply = selectedNew.size > 0 || selectedRemoved.size > 0

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50" onClick={onCancel} />
      <div className="relative bg-rook-card border border-rook-border rounded-lg p-4 max-w-md w-full mx-4 shadow-xl">
        <h3 className="text-sm font-semibold text-rook-text mb-3">Re-scan Results</h3>

        {diff.newServices.length > 0 && (
          <div className="mb-3">
            <div className="flex justify-between items-center mb-1">
              <span className="text-xs text-rook-muted">New services ({diff.newServices.length})</span>
              <button onClick={selectAllNew} className="text-[10px] text-rook-accent hover:underline">Select all</button>
            </div>
            <div className="space-y-1">
              {diff.newServices.map(svc => (
                <label key={svc.name} className="flex items-center gap-2 text-xs text-rook-text-secondary cursor-pointer">
                  <input
                    type="checkbox"
                    checked={selectedNew.has(svc.name)}
                    onChange={() => toggleNew(svc.name)}
                    className="rounded border-rook-border"
                  />
                  <span className="font-medium">{svc.name}</span>
                  <span className="text-rook-muted">{svc.image || `build: ${svc.build}`}</span>
                </label>
              ))}
            </div>
          </div>
        )}

        {diff.removedServices.length > 0 && (
          <div className="mb-3">
            <div className="flex justify-between items-center mb-1">
              <span className="text-xs text-rook-muted">Removed services ({diff.removedServices.length})</span>
              <button onClick={selectAllRemoved} className="text-[10px] text-rook-accent hover:underline">Select all</button>
            </div>
            <div className="space-y-1">
              {diff.removedServices.map(svc => (
                <label key={svc.name} className="flex items-center gap-2 text-xs text-rook-text-secondary cursor-pointer">
                  <input
                    type="checkbox"
                    checked={selectedRemoved.has(svc.name)}
                    onChange={() => toggleRemoved(svc.name)}
                    className="rounded border-rook-border"
                  />
                  <span className="font-medium">{svc.name}</span>
                  <span className="text-rook-muted">({svc.reason})</span>
                </label>
              ))}
            </div>
          </div>
        )}

        <div className="flex justify-end gap-2 mt-4">
          <button
            onClick={onCancel}
            className="px-3 py-1.5 text-xs rounded bg-rook-bg border border-rook-border text-rook-text-secondary hover:bg-rook-border/50"
          >
            Cancel
          </button>
          <button
            onClick={() => onApply(Array.from(selectedNew), Array.from(selectedRemoved))}
            disabled={!canApply}
            className="px-3 py-1.5 text-xs rounded bg-rook-accent hover:bg-rook-accent/80 text-white disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Apply Changes
          </button>
        </div>
      </div>
    </div>
  )
}
```

- [ ] **Step 2: Commit**
Run: `git add cmd/rook-gui/frontend/src/components/DiscoverDiffDialog.tsx && git commit -m "feat(gui): add DiscoverDiffDialog for selective merge"`

---

## Task 8: Frontend - Add Context Menu to Sidebar

**Files:**
- Modify: `cmd/rook-gui/frontend/src/components/Sidebar.tsx`

- [ ] **Step 1: Update Sidebar with context menu**

```tsx
// File: cmd/rook-gui/frontend/src/components/Sidebar.tsx
import { useState } from 'react'
import { WorkspaceInfo } from '../hooks/useWails'
import { StatusDot } from './StatusDot'
import { ContextMenu } from './ContextMenu'

interface SidebarProps {
  workspaces: WorkspaceInfo[]
  selected: string | null
  onSelect: (name: string | null) => void
  onAddWorkspace: () => void
  onRescan: (name: string) => void
  onRemove: (name: string) => void
}

function getWorkspaceStatus(ws: WorkspaceInfo): 'running' | 'partial' | 'stopped' {
  if (ws.runningCount === 0) return 'stopped'
  if (ws.runningCount === ws.serviceCount) return 'running'
  return 'partial'
}

const borderColors: Record<string, string> = {
  running: 'border-l-rook-running',
  partial: 'border-l-rook-partial',
  stopped: 'border-l-transparent',
}

const statusText: Record<string, string> = {
  running: 'text-rook-running',
  partial: 'text-rook-partial',
  stopped: 'text-rook-muted',
}

export function Sidebar({ workspaces, selected, onSelect, onAddWorkspace, onRescan, onRemove }: SidebarProps) {
  const [contextMenu, setContextMenu] = useState<{ x: number; y: number; workspace: string } | null>(null)

  const handleContextMenu = (e: React.MouseEvent, workspace: string) => {
    e.preventDefault()
    setContextMenu({ x: e.clientX, y: e.clientY, workspace })
  }

  return (
    <aside className="w-[220px] bg-rook-sidebar border-r border-rook-border p-3 flex flex-col">
      <p className="text-[10px] uppercase tracking-wider text-rook-muted mb-3">Workspaces</p>
      <div className="flex-1 space-y-1.5">
        {workspaces.map((ws) => {
          const status = getWorkspaceStatus(ws)
          const isSelected = ws.name === selected
          return (
            <button
              key={ws.name}
              onClick={() => onSelect(ws.name)}
              onContextMenu={(e) => handleContextMenu(e, ws.name)}
              className={`w-full text-left rounded-md p-2.5 border-l-[3px] transition-colors ${borderColors[status]} ${isSelected ? 'bg-rook-input' : 'bg-transparent hover:bg-rook-input/50'}`}
            >
              <div className="text-rook-text font-semibold text-sm">{ws.name}</div>
              <div className="flex items-center gap-1 mt-0.5">
                <StatusDot status={status} />
                <span className={`text-[10px] ${statusText[status]}`}>
                  {ws.runningCount === 0 ? 'stopped' : `${ws.runningCount}/${ws.serviceCount} running`}
                </span>
              </div>
            </button>
          )
        })}
      </div>
      <button
        onClick={onAddWorkspace}
        className="border border-dashed border-rook-border rounded-md p-2.5 text-center text-rook-muted text-sm hover:border-rook-muted transition-colors"
      >
        + Add Workspace
      </button>

      {contextMenu && (
        <ContextMenu
          x={contextMenu.x}
          y={contextMenu.y}
          items={[
            { label: 'Re-scan for changes', onClick: () => onRescan(contextMenu.workspace) },
            { label: 'Remove workspace', onClick: () => onRemove(contextMenu.workspace), danger: true },
          ]}
          onClose={() => setContextMenu(null)}
        />
      )}
    </aside>
  )
}
```

- [ ] **Step 2: Commit**
Run: `git add cmd/rook-gui/frontend/src/components/Sidebar.tsx && git commit -m "feat(gui): add context menu to Sidebar for rescan and remove"`

---

## Task 9: Frontend - Update WorkspaceDetail with Re-scan and Tab Rename

**Files:**
- Modify: `cmd/rook-gui/frontend/src/pages/WorkspaceDetail.tsx`

- [ ] **Step 1: Update WorkspaceDetail**

Change line 13 from:
```tsx
type Tab = 'services' | 'logs' | 'environment' | 'builds' | 'settings'
```
to:
```tsx
type Tab = 'services' | 'logs' | 'environment' | 'builds' | 'manifest'
```

Add imports at the top:
```tsx
import { DiscoverDiffDialog } from '../components/DiscoverDiffDialog'
import type { DiscoverDiff } from '../hooks/useWails'
import { useToast } from '../hooks/useToast'
```

Add state variables after the existing useState declarations:
```tsx
const [discoverDiff, setDiscoverDiff] = useState<DiscoverDiff | null>(null)
const [showDiscoverDialog, setShowDiscoverDialog] = useState(false)
const [rescanning, setRescanning] = useState(false)
const { show: showToast } = useToast()
```

Add handleRescan function after handleRebuildConfirm:
```tsx
const handleRescan = async () => {
  setRescanning(true)
  try {
    const diff = await window.go.api.WorkspaceAPI.DiscoverWorkspace(name)
    if (!diff.hasChanges) {
      showToast('No changes detected', 'info')
    } else {
      setDiscoverDiff(diff)
      setShowDiscoverDialog(true)
    }
  } catch (e) {
    console.error('Rescan failed:', e)
    showToast('Failed to scan: ' + e, 'error')
  } finally {
    setRescanning(false)
  }
}

const handleApplyDiscovery = async (newServices: string[], removedServices: string[]) => {
  setShowDiscoverDialog(false)
  try {
    await window.go.api.WorkspaceAPI.ApplyDiscovery(name, newServices, removedServices)
    showToast('Changes applied', 'success')
    refresh()
  } catch (e) {
    console.error('Apply discovery failed:', e)
    showToast('Failed to apply changes: ' + e, 'error')
  }
}
```

Add re-scan button in the header div, after the Start/Stop button:
```tsx
<button
  onClick={handleRescan}
  disabled={rescanning}
  className="text-[11px] text-rook-muted hover:text-rook-text border border-rook-border px-2 py-1 rounded disabled:opacity-50"
>
  {rescanning ? 'Scanning...' : 'Re-scan'}
</button>
```

Change the tab button array from `['services', 'logs', 'environment', 'builds', 'settings']` to `['services', 'logs', 'environment', 'builds', 'manifest']`

Change the tab content check from `tab === 'settings'` to `tab === 'manifest'`

Add the DiscoverDiffDialog at the end, after the RebuildDialog:
```tsx
<DiscoverDiffDialog
  open={showDiscoverDialog}
  diff={discoverDiff}
  onApply={handleApplyDiscovery}
  onCancel={() => setShowDiscoverDialog(false)}
/>
```

- [ ] **Step 2: Commit**
Run: `git add cmd/rook-gui/frontend/src/pages/WorkspaceDetail.tsx && git commit -m "feat(gui): add re-scan button and rename settings tab to manifest"`

---

## Task 10: Frontend - Update Dashboard with Settings Toggle

**Files:**
- Modify: `cmd/rook-gui/frontend/src/pages/Dashboard.tsx`

- [ ] **Step 1: Update Dashboard with auto-rebuild toggle**

Add import:
```tsx
import { useToast } from '../hooks/useToast'
```

Add state and logic inside the Dashboard component:
```tsx
const { show: showToast } = useToast()

const handleToggleAutoRebuild = async (value: boolean) => {
  try {
    await window.go.api.WorkspaceAPI.SaveSettings({ autoRebuild: value })
  } catch (e) {
    console.error('Failed to save settings:', e)
    showToast('Failed to save settings', 'error')
  }
}
```

Add a settings section after the Reset Ports button (before the ConfirmDialog):
```tsx
<div className="mt-6 pt-4 border-t border-rook-border">
  <label className="flex items-center gap-2 cursor-pointer">
    <input
      type="checkbox"
      defaultChecked={true}
      onChange={(e) => handleToggleAutoRebuild(e.target.checked)}
      className="rounded border-rook-border"
    />
    <span className="text-xs text-rook-text-secondary">Auto-rebuild on stale</span>
  </label>
</div>
```

- [ ] **Step 2: Commit**
Run: `git add cmd/rook-gui/frontend/src/pages/Dashboard.tsx && git commit -m "feat(gui): add auto-rebuild toggle to Dashboard"`

---

## Task 11: Frontend - Add ToastProvider to App

**Files:**
- Modify: `cmd/rook-gui/frontend/src/App.tsx`

- [ ] **Step 1: Wrap App with ToastProvider**

Add import:
```tsx
import { ToastProvider } from './hooks/useToast'
```

Wrap the main content with ToastProvider. The return should look like:
```tsx
return (
  <ToastProvider>
    <div className="flex h-screen bg-rook-bg text-rook-text">
      <Sidebar
        workspaces={workspaces}
        selected={selectedWorkspace}
        onSelect={setSelectedWorkspace}
        onAddWorkspace={() => setShowAddWizard(true)}
        onRescan={handleRescan}
        onRemove={handleRemoveWorkspace}
      />
      <main className="flex-1 overflow-hidden">
        {content}
      </main>
    </div>
    {showAddWizard && <DiscoveryWizard onClose={() => setShowAddWizard(false)} />}
  </ToastProvider>
)
```

Add handlers for rescan and remove in the App component (before the return):
```tsx
const { show: showToast } = useToast()

const handleRescan = async (name: string) => {
  try {
    const diff = await window.go.api.WorkspaceAPI.DiscoverWorkspace(name)
    if (!diff.hasChanges) {
      showToast('No changes detected', 'info')
    } else {
      setSelectedWorkspace(name)
    }
  } catch (e) {
    console.error('Rescan failed:', e)
    showToast('Failed to scan: ' + e, 'error')
  }
}

const handleRemoveWorkspace = async (name: string) => {
  try {
    await window.go.api.WorkspaceAPI.RemoveWorkspace(name)
    if (selectedWorkspace === name) {
      setSelectedWorkspace(null)
    }
    showToast('Workspace removed', 'success')
  } catch (e) {
    console.error('Remove failed:', e)
    showToast('Failed to remove workspace: ' + e, 'error')
  }
}
```

- [ ] **Step 2: Commit**
Run: `git add cmd/rook-gui/frontend/src/App.tsx && git commit -m "feat(gui): add ToastProvider and workspace action handlers to App"`

---

## Task 12: Frontend - Update Sidebar Props in App

**Files:**
- Modify: `cmd/rook-gui/frontend/src/App.tsx`

This is already done in Task 11 if the Sidebar component was updated with the new props.

- [ ] **Step 1: Verify build succeeds**
Run: `cd cmd/rook-gui/frontend && npm run build`
Expected: No errors

- [ ] **Step 2: Commit (if needed)**
Run: `git add cmd/rook-gui/frontend/src/App.tsx && git commit -m "fix(gui): wire up Sidebar rescan and remove handlers"`

---

## Task 13: Build and Test GUI

**Files:**
- None (verification task)

- [ ] **Step 1: Build the GUI**
Run: `make build-gui`
Expected: Binary created at `bin/rook-gui`

- [ ] **Step 2: Run API tests**
Run: `go test ./internal/api/ -v`
Expected: All tests pass

- [ ] **Step 3: Run all tests**
Run: `make test`
Expected: All tests pass

---

## Summary

This plan adds the following features to achieve GUI CLI parity for Phase 2:

1. **Re-scan workflow**: Detects new/removed services and shows a dialog for selective merge
2. **Workspace removal**: Right-click context menu in Sidebar with confirmation
3. **Global settings UI**: Auto-rebuild toggle on Dashboard
4. **Tab rename**: "Settings" → "Manifest" in WorkspaceDetail

All changes follow TDD principles with tests written first, and maintain consistency with the existing codebase patterns.
