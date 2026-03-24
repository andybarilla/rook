import { useState, useEffect } from 'react'
import type { DiscoverDiff } from '../hooks/useWails'

interface DiscoverDiffDialogProps {
  open: boolean
  diff: DiscoverDiff | null
  onApply: (newServices: string[], removedServices: string[]) => void
  onCancel: () => void
}

export function DiscoverDiffDialog({ open, diff, onApply, onCancel }: DiscoverDiffDialogProps) {
  const [selectedNew, setSelectedNew] = useState<Set<string>>(new Set())
  const [selectedRemoved, setSelectedRemoved] = useState<Set<string>>(new Set())

  // Reset selections when diff changes — all checked by default
  useEffect(() => {
    if (diff) {
      setSelectedNew(new Set(diff.newServices.map(s => s.name)))
      setSelectedRemoved(new Set(diff.removedServices.map(s => s.name)))
    }
  }, [diff])

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

  const canApply = selectedNew.size > 0 || selectedRemoved.size > 0

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50" onClick={onCancel} />
      <div className="relative bg-rook-card border border-rook-border rounded-lg p-4 max-w-md w-full mx-4 shadow-xl">
        <h3 className="text-sm font-semibold text-rook-text mb-3">Re-scan Results</h3>

        {diff.newServices.length > 0 && (
          <div className="mb-3">
            <span className="text-xs text-rook-muted">New services ({diff.newServices.length})</span>
            <div className="space-y-1 mt-1">
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
            <span className="text-xs text-rook-muted">Removed services ({diff.removedServices.length})</span>
            <div className="space-y-1 mt-1">
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
            className="px-3 py-1.5 text-xs rounded bg-rook-active hover:bg-rook-active-hover text-white disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Apply Changes
          </button>
        </div>
      </div>
    </div>
  )
}
