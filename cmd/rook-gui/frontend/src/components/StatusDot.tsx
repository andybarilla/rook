interface StatusDotProps {
  status: 'running' | 'starting' | 'crashed' | 'stopped' | 'partial'
  size?: 'sm' | 'md'
}

const colorMap: Record<string, string> = {
  running: 'bg-rook-running',
  starting: 'bg-rook-partial',
  partial: 'bg-rook-partial',
  crashed: 'bg-rook-crashed',
  stopped: 'bg-rook-stopped',
}

export function StatusDot({ status, size = 'sm' }: StatusDotProps) {
  const sizeClass = size === 'sm' ? 'w-1.5 h-1.5' : 'w-2 h-2'
  return <div className={`${sizeClass} rounded-full ${colorMap[status] || colorMap.stopped}`} />
}
