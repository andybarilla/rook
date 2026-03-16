import { useCallback, useEffect, useState } from 'react'
import type { WorkspaceDetail as WorkspaceDetailType } from '../hooks/useWails'
import { ServiceList } from '../components/ServiceList'
import { ProfileSwitcher } from '../components/ProfileSwitcher'
import { LogViewer } from '../components/LogViewer'
import { EnvViewer } from '../components/EnvViewer'
import { ManifestEditor } from '../components/ManifestEditor'

interface WorkspaceDetailProps { name: string }

type Tab = 'services' | 'logs' | 'environment' | 'settings'

export function WorkspaceDetail({ name }: WorkspaceDetailProps) {
  const [detail, setDetail] = useState<WorkspaceDetailType | null>(null)
  const [tab, setTab] = useState<Tab>('services')

  const refresh = useCallback(() => {
    window.go.api.WorkspaceAPI.GetWorkspace(name).then(setDetail).catch(console.error)
  }, [name])

  useEffect(() => {
    refresh()
    const off = window.runtime.EventsOn('service:status', () => refresh())
    return off
  }, [name, refresh])

  if (!detail) return <div className="p-4 text-rook-muted">Loading...</div>

  const hasRunning = detail.services.some(s => s.status === 'running' || s.status === 'starting')

  return (
    <div className="flex flex-col h-full">
      <div className="px-4 py-3 border-b border-rook-border flex justify-between items-center">
        <div>
          <h1 className="text-base font-semibold text-rook-text">{detail.name}</h1>
          <p className="text-[10px] text-rook-muted">{detail.path}</p>
        </div>
        <div className="flex items-center gap-2">
          <ProfileSwitcher profiles={detail.profiles} active={detail.activeProfile}
            onChange={(p) => window.go.api.WorkspaceAPI.StartWorkspace(name, p).then(refresh)} />
          {hasRunning ? (
            <button onClick={() => window.go.api.WorkspaceAPI.StopWorkspace(name).then(refresh)}
              className="bg-rook-crashed text-white px-3 py-1 rounded text-[11px]">Stop All</button>
          ) : (
            <button onClick={() => window.go.api.WorkspaceAPI.StartWorkspace(name, detail.profiles[0] || 'all').then(refresh)}
              className="bg-rook-running text-rook-bg px-3 py-1 rounded text-[11px] font-semibold">Start</button>
          )}
        </div>
      </div>
      <div className="flex border-b border-rook-border">
        {(['services', 'logs', 'environment', 'settings'] as Tab[]).map(t => (
          <button key={t} onClick={() => setTab(t)}
            className={`px-4 py-2 text-[11px] border-b-2 capitalize ${tab === t ? 'text-rook-text border-rook-accent font-semibold' : 'text-rook-muted border-transparent'}`}>
            {t}
          </button>
        ))}
      </div>
      <div className="flex-1 overflow-auto">
        {tab === 'services' && (
          <div className="p-3">
            <ServiceList services={detail.services} workspaceName={name}
              onStart={(svc) => window.go.api.WorkspaceAPI.StartService(name, svc).then(refresh)}
              onStop={(svc) => window.go.api.WorkspaceAPI.StopService(name, svc).then(refresh)}
              onRestart={(svc) => window.go.api.WorkspaceAPI.RestartService(name, svc).then(refresh)} />
          </div>
        )}
        {tab === 'logs' && <LogViewer workspaceName={name} services={detail.services.map(s => s.name)} />}
        {tab === 'environment' && <EnvViewer workspaceName={name} />}
        {tab === 'settings' && <ManifestEditor workspaceName={name} />}
      </div>
    </div>
  )
}
