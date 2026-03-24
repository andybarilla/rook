interface BuildStatusBadgeProps {
  status: 'up_to_date' | 'needs_rebuild' | 'no_build_context'
  reason?: string
}

export function BuildStatusBadge({ status, reason }: BuildStatusBadgeProps) {
  if (status !== 'needs_rebuild') {
    return null
  }

  return (
    <span
      className="inline-flex items-center px-1.5 py-0.5 text-[9px] font-medium bg-rook-attention/20 text-rook-attention border border-rook-attention/30"
      title={reason || 'Needs rebuild'}
    >
      build
    </span>
  )
}
