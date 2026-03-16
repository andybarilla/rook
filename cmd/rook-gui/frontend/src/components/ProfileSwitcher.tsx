interface ProfileSwitcherProps {
  profiles: string[]
  active: string | undefined
  onChange: (profile: string) => void
}

export function ProfileSwitcher({ profiles, active, onChange }: ProfileSwitcherProps) {
  return (
    <select
      value={active || ''}
      onChange={(e) => onChange(e.target.value)}
      className="bg-rook-input text-rook-text-secondary border border-rook-border rounded px-2 py-1 text-[11px]"
    >
      {!active && <option value="">stopped</option>}
      {profiles.map((p) => (
        <option key={p} value={p}>{p}</option>
      ))}
    </select>
  )
}
