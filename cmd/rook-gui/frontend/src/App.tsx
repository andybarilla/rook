import { useState } from 'react'
import { Sidebar } from './components/Sidebar'
import { Dashboard } from './pages/Dashboard'
import { useWorkspaces } from './hooks/useWails'

function App() {
  const { workspaces, refresh } = useWorkspaces()
  const [selected, setSelected] = useState<string | null>(null)
  const [showWizard, setShowWizard] = useState(false)

  return (
    <div className="flex h-screen bg-rook-bg text-rook-text">
      <Sidebar
        workspaces={workspaces}
        selected={selected}
        onSelect={setSelected}
        onAddWorkspace={() => setShowWizard(true)}
      />
      <main className="flex-1 overflow-auto">
        {selected === null ? (
          <Dashboard workspaces={workspaces} />
        ) : (
          <div className="p-4">
            <h1 className="text-lg font-semibold">{selected}</h1>
            <p className="text-sm text-rook-muted">Workspace detail coming next...</p>
          </div>
        )}
      </main>
    </div>
  )
}

export default App
