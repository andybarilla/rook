import { ServiceInfo } from '../hooks/useWails'
import { StatusDot } from './StatusDot'

interface ServiceListProps {
  services: ServiceInfo[]
  workspaceName: string
  onStart: (service: string) => void
  onStop: (service: string) => void
  onRestart: (service: string) => void
}

export function ServiceList({ services, onStart, onStop, onRestart }: ServiceListProps) {
  return (
    <div className="space-y-1.5">
      {services.map((svc) => (
        <div key={svc.name} className="bg-rook-card rounded-md px-3 py-2.5 flex justify-between items-center">
          <div className="flex items-center gap-2">
            <StatusDot status={svc.status} size="md" />
            <div>
              <div className="text-rook-text font-semibold text-sm">{svc.name}</div>
              <div className="text-rook-muted text-[10px]">{svc.image || `${svc.command} (process)`}</div>
            </div>
          </div>
          <div className="flex items-center gap-2.5">
            {svc.port ? <span className="text-rook-text-secondary text-[10px] font-mono">:{svc.port}</span> : null}
            {svc.status === 'running' || svc.status === 'starting' ? (
              <>
                <ActionLink label="restart" onClick={() => onRestart(svc.name)} />
                <ActionLink label="stop" onClick={() => onStop(svc.name)} color="text-rook-crashed" />
              </>
            ) : (
              <ActionLink label="start" onClick={() => onStart(svc.name)} color="text-rook-running" />
            )}
          </div>
        </div>
      ))}
    </div>
  )
}

function ActionLink({ label, onClick, color = 'text-rook-muted' }: { label: string; onClick: () => void; color?: string }) {
  return (
    <button onClick={onClick} className={`${color} text-[10px] hover:underline cursor-pointer bg-transparent border-none`}>
      {label}
    </button>
  )
}
