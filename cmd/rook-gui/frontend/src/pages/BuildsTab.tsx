import { useEffect, useState } from 'react'
import type { BuildStatus, BuildCheckResult } from '../hooks/useWails'

interface BuildsTabProps {
  workspaceName: string
}

export function BuildsTab({ workspaceName }: BuildsTabProps) {
  const [result, setResult] = useState<BuildCheckResult | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const refresh = async () => {
    setLoading(true)
    setError(null)
    try {
      const r = await window.go.api.WorkspaceAPI.CheckBuilds(workspaceName)
      setResult(r)
    } catch (e) {
      setError(String(e))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    refresh()
  }, [workspaceName])

  if (loading) {
    return <div className="p-4 text-rook-muted text-xs">Checking builds...</div>
  }

  if (error) {
    return <div className="p-4 text-rook-crashed text-xs">Error: {error}</div>
  }

  if (!result || result.services.length === 0) {
    return <div className="p-4 text-rook-muted text-xs">No services in workspace</div>
  }

  return (
    <div className="p-3">
      <div className="flex justify-between items-center mb-3">
        <h3 className="text-xs font-semibold text-rook-text">Build Status</h3>
        <button
          onClick={refresh}
          className="text-[10px] text-rook-muted hover:text-rook-text"
        >
          Refresh
        </button>
      </div>
      <div className="space-y-1.5">
        {result.services.map(svc => (
          <div
            key={svc.name}
            className="bg-rook-card rounded-md px-3 py-2.5 flex justify-between items-center"
          >
            <div className="flex items-center gap-2">
              <StatusIcon status={svc.status} />
              <div>
                <div className="text-rook-text font-semibold text-sm">{svc.name}</div>
                <StatusText status={svc.status} reasons={svc.reasons} />
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

function StatusIcon({ status }: { status: BuildStatus['status'] }) {
  switch (status) {
    case 'up_to_date':
      return <span className="text-rook-running">✅</span>
    case 'needs_rebuild':
      return <span className="text-orange-400">⚠️</span>
    default:
      return <span className="text-rook-muted">○</span>
  }
}

function StatusText({
  status,
  reasons,
}: {
  status: BuildStatus['status']
  reasons?: string[]
}) {
  switch (status) {
    case 'up_to_date':
      return <div className="text-rook-muted text-[10px]">Up to date</div>
    case 'needs_rebuild':
      return (
        <div className="text-orange-400 text-[10px]">
          Needs rebuild{reasons && reasons.length > 0 && ` (${reasons[0]})`}
        </div>
      )
    default:
      return <div className="text-rook-muted text-[10px]">No build context</div>
  }
}
