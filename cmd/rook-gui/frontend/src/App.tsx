import { useState } from 'react'
import { Sidebar } from './components/Sidebar'
import { Dashboard } from './pages/Dashboard'
import { WorkspaceDetail } from './pages/WorkspaceDetail'
import { DiscoveryWizard } from './components/DiscoveryWizard'
import { useWorkspaces } from './hooks/useWails'
import { SettingsProvider } from './hooks/useSettings'

function App() {
  const { workspaces, refresh } = useWorkspaces()
  const [selected, setSelected] = useState<string | null>(null)
  const [showWizard, setShowWizard] = useState(false)

  return (
    <SettingsProvider>
      <div className="flex h-screen bg-rook-bg text-rook-text">
        <Sidebar workspaces={workspaces} selected={selected} onSelect={setSelected} onAddWorkspace={() => setShowWizard(true)} />
        <main className="flex-1 overflow-auto">
          {selected === null ? <Dashboard workspaces={workspaces} /> : <WorkspaceDetail name={selected} />}
        </main>
        {showWizard && <DiscoveryWizard onClose={() => setShowWizard(false)} onComplete={() => refresh()} />}
      </div>
    </SettingsProvider>
  )
}

export default App
