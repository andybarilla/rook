import { useEffect, useRef } from 'react'

interface ContextMenuItem {
  label: string
  onClick: () => void
  danger?: boolean
}

interface ContextMenuProps {
  x: number
  y: number
  items: ContextMenuItem[]
  onClose: () => void
}

export function ContextMenu({ x, y, items, onClose }: ContextMenuProps) {
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const handleClick = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        onClose()
      }
    }
    document.addEventListener('click', handleClick)
    return () => document.removeEventListener('click', handleClick)
  }, [onClose])

  return (
    <div
      ref={ref}
      className="fixed bg-rook-card border border-rook-border rounded-md shadow-lg py-1 z-50"
      style={{ left: x, top: y }}
    >
      {items.map((item, i) => (
        <button
          key={i}
          onClick={() => {
            item.onClick()
            onClose()
          }}
          className={`block w-full text-left px-3 py-1.5 text-xs ${
            item.danger ? 'text-rook-error hover:bg-rook-error/10' : 'text-rook-text hover:bg-rook-border/50'
          }`}
        >
          {item.label}
        </button>
      ))}
    </div>
  )
}
