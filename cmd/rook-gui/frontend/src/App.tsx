import { useState } from 'react'
import { Sidebar } from './components/Sidebar'
import { Dashboard } from './pages/Dashboard'
import { WorkspaceDetail } from './pages/WorkspaceDetail'
import { DiscoveryWizard } from './components/DiscoveryWizard'
import { ConfirmDialog } from './components/ConfirmDialog'
import { useWorkspaces } from './hooks/useWails'
import { SettingsProvider } from './hooks/useSettings'
import { ToastProvider, useToast } from './hooks/useToast'

function AppContent() {
  const { workspaces, refresh } = useWorkspaces()
  const [selected, setSelected] = useState<string | null>(null)
  const [showWizard, setShowWizard] = useState(false)
  const [removeTarget, setRemoveTarget] = useState<string | null>(null)
  const { show: showToast } = useToast()

  const handleRescan = (name: string) => {
    // Navigate to the workspace — the re-scan button is in WorkspaceDetail header
    setSelected(name)
  }

  const handleRemove = async () => {
    if (!removeTarget) return
    try {
      await window.go.api.WorkspaceAPI.RemoveWorkspace(removeTarget)
      if (selected === removeTarget) {
        setSelected(null)
      }
      showToast('Workspace removed', 'success')
      refresh()
    } catch (e) {
      console.error('Remove failed:', e)
      showToast('Failed to remove workspace: ' + e, 'error')
    }
    setRemoveTarget(null)
  }

  return (
    <div className="flex h-screen bg-rook-bg text-rook-text">
      <Sidebar
        workspaces={workspaces}
        selected={selected}
        onSelect={setSelected}
        onAddWorkspace={() => setShowWizard(true)}
        onRescan={handleRescan}
        onRemove={(name) => setRemoveTarget(name)}
      />
      <main className="flex-1 overflow-auto">
        {selected === null ? <Dashboard workspaces={workspaces} /> : <WorkspaceDetail name={selected} />}
      </main>
      {showWizard && <DiscoveryWizard onClose={() => setShowWizard(false)} onComplete={() => refresh()} />}

      <ConfirmDialog
        open={removeTarget !== null}
        title="Remove workspace?"
        message={`This will stop all services and unregister '${removeTarget}'. The project files will not be deleted.`}
        confirmLabel="Remove"
        variant="danger"
        onConfirm={handleRemove}
        onCancel={() => setRemoveTarget(null)}
      />
    </div>
  )
}

function App() {
  return (
    <SettingsProvider>
      <ToastProvider>
        <AppContent />
      </ToastProvider>
    </SettingsProvider>
  )
}

export default App
