import { useEffect, useState } from 'react'

interface EnvViewerProps { workspaceName: string }

interface EnvVar { key: string; template: string; resolved: string }

export function EnvViewer({ workspaceName }: EnvViewerProps) {
  const [envData, setEnvData] = useState<Record<string, EnvVar[]>>({})
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    window.go.api.WorkspaceAPI.GetEnv(workspaceName)
      .then(data => setEnvData(data || {}))
      .catch((e: Error) => setError(e.message))
  }, [workspaceName])

  if (error) return <div className="p-4 text-rook-muted text-sm">{error}</div>

  const serviceNames = Object.keys(envData)
  if (serviceNames.length === 0) return <div className="p-4 text-rook-muted text-sm">No environment variables defined.</div>

  return (
    <div className="p-4 space-y-4">
      {serviceNames.map(svc => (
        <div key={svc}>
          <h3 className="text-xs uppercase tracking-wider text-rook-text-secondary mb-2">{svc}</h3>
          <div className="bg-rook-card rounded-md overflow-hidden text-xs">
            <div className="grid grid-cols-[1fr_2fr_2fr] px-3 py-1.5 text-rook-muted border-b border-rook-border">
              <span>Key</span><span>Template</span><span>Resolved</span>
            </div>
            {envData[svc].map((v, i, arr) => (
              <div key={v.key} className={`grid grid-cols-[1fr_2fr_2fr] px-3 py-1.5 ${i < arr.length - 1 ? 'border-b border-rook-border' : ''}`}>
                <span className="text-rook-text font-mono">{v.key}</span>
                <span className="text-rook-muted font-mono break-all">{v.template}</span>
                <span className="text-rook-text-secondary font-mono break-all">{v.resolved}</span>
              </div>
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}
