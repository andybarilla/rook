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
      className="inline-flex items-center px-1.5 py-0.5 rounded text-[9px] font-medium bg-orange-500/20 text-orange-400 border border-orange-500/30"
      title={reason || 'Needs rebuild'}
    >
      build
    </span>
  )
}
