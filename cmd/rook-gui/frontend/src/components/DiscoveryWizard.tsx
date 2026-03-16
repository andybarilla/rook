import { useState } from 'react'

interface DiscoveryWizardProps {
  onClose: () => void
  onComplete: () => void
}

export function DiscoveryWizard({ onClose, onComplete }: DiscoveryWizardProps) {
  const [path, setPath] = useState('')
  const [status, setStatus] = useState<'input' | 'discovering' | 'done' | 'error'>('input')
  const [result, setResult] = useState<any>(null)
  const [error, setError] = useState<string | null>(null)

  const handleDiscover = async () => {
    if (!path.trim()) return
    setStatus('discovering')
    try {
      const r = await window.go.api.WorkspaceAPI.AddWorkspace(path.trim())
      setResult(r)
      setStatus('done')
    } catch (e: any) {
      setError(e.message || 'Discovery failed')
      setStatus('error')
    }
  }

  return (
    <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
      <div className="bg-rook-bg border border-rook-border rounded-lg w-[500px] max-h-[80vh] overflow-auto">
        <div className="flex justify-between items-center px-4 py-3 border-b border-rook-border">
          <h2 className="text-sm font-semibold text-rook-text">Add Workspace</h2>
          <button onClick={onClose} className="text-rook-muted hover:text-rook-text text-lg">×</button>
        </div>
        <div className="p-4">
          {status === 'input' && (
            <>
              <label className="text-xs text-rook-text-secondary block mb-1">Project directory</label>
              <input type="text" value={path} onChange={e => setPath(e.target.value)}
                placeholder="/home/user/dev/my-project"
                className="w-full bg-rook-input text-rook-text border border-rook-border rounded px-3 py-2 text-sm mb-3"
                onKeyDown={e => e.key === 'Enter' && handleDiscover()} />
              <button onClick={handleDiscover}
                className="bg-rook-accent text-white px-4 py-2 rounded text-sm w-full">
                Discover & Add
              </button>
            </>
          )}
          {status === 'discovering' && (
            <p className="text-rook-muted text-sm text-center py-8">Discovering services...</p>
          )}
          {status === 'done' && result && (
            <>
              <p className="text-rook-running text-sm mb-3">Workspace added!</p>
              {result.source && <p className="text-rook-muted text-xs mb-2">Discovered from: {result.source}</p>}
              {result.services && Object.keys(result.services).length > 0 && (
                <div className="space-y-1 mb-4">
                  {Object.entries(result.services).map(([name, svc]: [string, any]) => (
                    <div key={name} className="bg-rook-card rounded px-3 py-2 text-xs">
                      <span className="text-rook-text font-semibold">{name}</span>
                      <span className="text-rook-muted ml-2">{svc.image || svc.command || 'service'}</span>
                    </div>
                  ))}
                </div>
              )}
              <button onClick={() => { onComplete(); onClose() }}
                className="bg-rook-accent text-white px-4 py-2 rounded text-sm w-full">Done</button>
            </>
          )}
          {status === 'error' && (
            <>
              <p className="text-rook-crashed text-sm mb-3">{error}</p>
              <button onClick={() => { setStatus('input'); setError(null) }}
                className="text-rook-muted text-xs hover:underline">Try again</button>
            </>
          )}
        </div>
      </div>
    </div>
  )
}
