import { useCallback, useEffect, useState } from 'react'
import type { WorkspaceDetail as WorkspaceDetailType, BuildCheckResult, Settings } from '../hooks/useWails'
import { ServiceList } from '../components/ServiceList'
import { ProfileSwitcher } from '../components/ProfileSwitcher'
import { LogViewer } from '../components/LogViewer'
import { EnvViewer } from '../components/EnvViewer'
import { ManifestEditor } from '../components/ManifestEditor'
import { BuildsTab } from './BuildsTab'
import { RebuildDialog } from '../components/RebuildDialog'

interface WorkspaceDetailProps { name: string }

type Tab = 'services' | 'logs' | 'environment' | 'builds' | 'settings'

export function WorkspaceDetail({ name }: WorkspaceDetailProps) {
  const [detail, setDetail] = useState<WorkspaceDetailType | null>(null)
  const [tab, setTab] = useState<Tab>('services')
  const [settings, setSettings] = useState<Settings>({ autoRebuild: true })
  const [buildResult, setBuildResult] = useState<BuildCheckResult | null>(null)
  const [showRebuildDialog, setShowRebuildDialog] = useState(false)
  const [pendingStart, setPendingStart] = useState<{ profile: string } | null>(null)
  const [starting, setStarting] = useState(false)

  const refresh = useCallback(() => {
    window.go.api.WorkspaceAPI.GetWorkspace(name).then(setDetail).catch(console.error)
  }, [name])

  const refreshSettings = useCallback(async () => {
    try {
      const s = await window.go.api.WorkspaceAPI.GetSettings()
      setSettings(s || { autoRebuild: true })
    } catch (e) {
      console.error('Failed to get settings:', e)
    }
  }, [])

  useEffect(() => {
    refresh()
    refreshSettings()
    const off = window.runtime.EventsOn('service:status', () => refresh())
    return off
  }, [name, refresh, refreshSettings])

  const handleStart = async (profile: string) => {
    setStarting(true)
    try {
      // Check for stale builds
      const result = await window.go.api.WorkspaceAPI.CheckBuilds(name)
      setBuildResult(result)

      if (result.hasStale) {
        if (settings.autoRebuild) {
          // Auto-rebuild enabled, just start with forceBuild=true
          await window.go.api.WorkspaceAPI.StartWorkspace(name, profile, true)
          refresh()
        } else {
          // Show rebuild dialog
          setPendingStart({ profile })
          setShowRebuildDialog(true)
        }
      } else {
        // No stale builds, start normally
        await window.go.api.WorkspaceAPI.StartWorkspace(name, profile, false)
        refresh()
      }
    } catch (e) {
      console.error('Start failed:', e)
    } finally {
      setStarting(false)
    }
  }

  const handleRebuildConfirm = async (forceBuild: boolean) => {
    if (!pendingStart) return
    setShowRebuildDialog(false)
    try {
      await window.go.api.WorkspaceAPI.StartWorkspace(name, pendingStart.profile, forceBuild)
      refresh()
    } catch (e) {
      console.error('Start failed:', e)
    }
    setPendingStart(null)
  }

  if (!detail) return <div className="p-4 text-rook-muted">Loading...</div>

  const hasRunning = detail.services.some(s => s.status === 'running' || s.status === 'starting')
  const activeProfile = detail.activeProfile || detail.profiles[0] || 'all'

  return (
    <div className="flex flex-col h-full">
      <div className="px-4 py-3 border-b border-rook-border flex justify-between items-center">
        <div>
          <h1 className="text-base font-semibold text-rook-text">{detail.name}</h1>
          <p className="text-[10px] text-rook-muted">{detail.path}</p>
        </div>
        <div className="flex items-center gap-2">
          <ProfileSwitcher profiles={detail.profiles} active={detail.activeProfile}
            onChange={(p) => handleStart(p)} />
          {hasRunning ? (
            <button onClick={() => window.go.api.WorkspaceAPI.StopWorkspace(name).then(refresh)}
              className="bg-rook-crashed text-white px-3 py-1 rounded text-[11px]">Stop All</button>
          ) : (
            <button
              onClick={() => handleStart(activeProfile)}
              disabled={starting}
              className="bg-rook-running text-rook-bg px-3 py-1 rounded text-[11px] font-semibold disabled:opacity-50"
            >
              {starting ? 'Starting...' : 'Start'}
            </button>
          )}
        </div>
      </div>
      <div className="flex border-b border-rook-border">
        {(['services', 'logs', 'environment', 'builds', 'settings'] as Tab[]).map(t => (
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
        {tab === 'builds' && <BuildsTab workspaceName={name} />}
        {tab === 'settings' && <ManifestEditor workspaceName={name} />}
      </div>

      <RebuildDialog
        open={showRebuildDialog}
        services={buildResult?.services || []}
        onRebuild={() => handleRebuildConfirm(true)}
        onSkip={() => handleRebuildConfirm(false)}
        onCancel={() => {
          setShowRebuildDialog(false)
          setPendingStart(null)
        }}
      />
    </div>
  )
}
