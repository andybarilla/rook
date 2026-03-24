interface StatusDotProps {
  status: 'running' | 'starting' | 'crashed' | 'stopped' | 'partial'
  size?: 'sm' | 'md'
}

const colorMap: Record<string, string> = {
  running: 'bg-rook-active',
  starting: 'bg-rook-attention',
  partial: 'bg-rook-attention',
  crashed: 'bg-rook-error',
  stopped: 'bg-rook-idle',
}

export function StatusDot({ status, size = 'sm' }: StatusDotProps) {
  const sizeClass = size === 'sm' ? 'w-1.5 h-1.5' : 'w-2 h-2'
  return <div className={`${sizeClass} rounded-full ${colorMap[status] || colorMap.stopped}`} />
}
