import { WorkspaceInfo } from '../hooks/useWails'
import { StatusDot } from './StatusDot'

interface SidebarProps {
  workspaces: WorkspaceInfo[]
  selected: string | null
  onSelect: (name: string | null) => void
  onAddWorkspace: () => void
}

function getWorkspaceStatus(ws: WorkspaceInfo): 'running' | 'partial' | 'stopped' {
  if (ws.runningCount === 0) return 'stopped'
  if (ws.runningCount === ws.serviceCount) return 'running'
  return 'partial'
}

const borderColors: Record<string, string> = {
  running: 'border-l-rook-running',
  partial: 'border-l-rook-partial',
  stopped: 'border-l-transparent',
}

const statusText: Record<string, string> = {
  running: 'text-rook-running',
  partial: 'text-rook-partial',
  stopped: 'text-rook-muted',
}

export function Sidebar({ workspaces, selected, onSelect, onAddWorkspace }: SidebarProps) {
  return (
    <aside className="w-[220px] bg-rook-sidebar border-r border-rook-border p-3 flex flex-col">
      <p className="text-[10px] uppercase tracking-wider text-rook-muted mb-3">Workspaces</p>
      <div className="flex-1 space-y-1.5">
        {workspaces.map((ws) => {
          const status = getWorkspaceStatus(ws)
          const isSelected = ws.name === selected
          return (
            <button
              key={ws.name}
              onClick={() => onSelect(ws.name)}
              className={`w-full text-left rounded-md p-2.5 border-l-[3px] transition-colors ${borderColors[status]} ${isSelected ? 'bg-rook-input' : 'bg-transparent hover:bg-rook-input/50'}`}
            >
              <div className="text-rook-text font-semibold text-sm">{ws.name}</div>
              <div className="flex items-center gap-1 mt-0.5">
                <StatusDot status={status} />
                <span className={`text-[10px] ${statusText[status]}`}>
                  {ws.runningCount === 0 ? 'stopped' : `${ws.runningCount}/${ws.serviceCount} running`}
                </span>
              </div>
            </button>
          )
        })}
      </div>
      <button
        onClick={onAddWorkspace}
        className="border border-dashed border-rook-border rounded-md p-2.5 text-center text-rook-muted text-sm hover:border-rook-muted transition-colors"
      >
        + Add Workspace
      </button>
    </aside>
  )
}
