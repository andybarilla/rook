interface ManifestEditorProps { workspaceName: string }

export function ManifestEditor({ workspaceName }: ManifestEditorProps) {
  return (
    <div className="p-4">
      <h3 className="text-xs uppercase tracking-wider text-rook-text-secondary mb-2">Manifest (rook.yaml)</h3>
      <div className="bg-rook-card p-3 text-rook-muted text-sm">
        <p>The visual manifest editor is planned for a future update.</p>
        <p className="mt-2">
          Edit <code className="text-rook-text-secondary">rook.yaml</code> directly in your project directory.
          Changes will be picked up on the next refresh.
        </p>
      </div>
    </div>
  )
}
