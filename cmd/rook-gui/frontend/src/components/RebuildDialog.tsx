import type { BuildStatus } from '../hooks/useWails'

interface RebuildDialogProps {
  open: boolean
  services: BuildStatus[]
  onRebuild: () => void
  onSkip: () => void
  onCancel: () => void
}

export function RebuildDialog({ open, services, onRebuild, onSkip, onCancel }: RebuildDialogProps) {
  if (!open) return null

  const staleServices = services.filter(s => s.status === 'needs_rebuild')

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50" onClick={onCancel} />
      <div className="relative bg-rook-card border border-rook-border p-4 max-w-md w-full mx-4 shadow-xl">
        <h3 className="text-sm font-semibold text-rook-text mb-2">Rebuild Required</h3>
        <p className="text-xs text-rook-muted mb-3">
          {staleServices.length} service(s) need rebuilding:
        </p>
        <ul className="text-xs text-rook-text-secondary mb-4 space-y-1 max-h-32 overflow-auto">
          {staleServices.map(s => (
            <li key={s.name} className="flex justify-between">
              <span className="font-medium">{s.name}</span>
              {s.reasons && s.reasons.length > 0 && (
                <span className="text-rook-muted">{s.reasons[0]}</span>
              )}
            </li>
          ))}
        </ul>
        <div className="flex justify-end gap-2">
          <button
            onClick={onCancel}
            className="px-3 py-1.5 text-xs bg-rook-bg border border-rook-border text-rook-text-secondary hover:bg-rook-border/50"
          >
            Cancel
          </button>
          <button
            onClick={onSkip}
            className="px-3 py-1.5 text-xs bg-rook-bg border border-rook-border text-rook-text-secondary hover:bg-rook-border/50"
          >
            Skip
          </button>
          <button
            onClick={onRebuild}
            className="px-3 py-1.5 text-xs bg-rook-active hover:bg-rook-active-hover text-white"
          >
            Rebuild
          </button>
        </div>
      </div>
    </div>
  )
}
