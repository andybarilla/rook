function App() {
  return (
    <div className="flex h-screen bg-rook-bg text-rook-text">
      <aside className="w-[220px] bg-rook-sidebar border-r border-rook-border p-3">
        <p className="text-xs uppercase tracking-wider text-rook-muted">Workspaces</p>
      </aside>
      <main className="flex-1 p-4">
        <h1 className="text-lg font-semibold">Dashboard</h1>
        <p className="text-sm text-rook-muted">Rook is running.</p>
      </main>
    </div>
  )
}

export default App
