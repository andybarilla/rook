interface ConfirmDialogProps {
  open: boolean
  title: string
  message: string
  confirmLabel?: string
  cancelLabel?: string
  variant?: 'default' | 'danger'
  onConfirm: () => void
  onCancel: () => void
}

export function ConfirmDialog({
  open,
  title,
  message,
  confirmLabel = 'Confirm',
  cancelLabel = 'Cancel',
  variant = 'default',
  onConfirm,
  onCancel,
}: ConfirmDialogProps) {
  if (!open) return null

  const confirmBtnClass =
    variant === 'danger'
      ? 'bg-rook-error hover:bg-rook-error/80 text-white'
      : 'bg-rook-active hover:bg-rook-active-hover text-white'

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50" onClick={onCancel} />
      <div className="relative bg-rook-card border border-rook-border rounded-lg p-4 max-w-md w-full mx-4 shadow-xl">
        <h3 className="text-sm font-semibold text-rook-text mb-2">{title}</h3>
        <p className="text-xs text-rook-muted mb-4">{message}</p>
        <div className="flex justify-end gap-2">
          <button
            onClick={onCancel}
            className="px-3 py-1.5 text-xs rounded bg-rook-bg border border-rook-border text-rook-text-secondary hover:bg-rook-border/50"
          >
            {cancelLabel}
          </button>
          <button
            onClick={onConfirm}
            className={`px-3 py-1.5 text-xs rounded ${confirmBtnClass}`}
          >
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>
  )
}
