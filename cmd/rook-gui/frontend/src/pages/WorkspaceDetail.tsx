import { useCallback, useEffect, useState } from 'react'
import type { WorkspaceDetail as WorkspaceDetailType, BuildCheckResult, DiscoverDiff } from '../hooks/useWails'
import { useSettings } from '../hooks/useSettings'
import { useToast } from '../hooks/useToast'
import { ServiceList } from '../components/ServiceList'
import { ProfileSwitcher } from '../components/ProfileSwitcher'
import { LogViewer } from '../components/LogViewer'
import { EnvViewer } from '../components/EnvViewer'
import { ManifestEditor } from '../components/ManifestEditor'
import { BuildsTab } from './BuildsTab'
import { RebuildDialog } from '../components/RebuildDialog'
import { DiscoverDiffDialog } from '../components/DiscoverDiffDialog'

interface WorkspaceDetailProps { name: string }

type Tab = 'services' | 'logs' | 'environment' | 'builds' | 'manifest'

export function WorkspaceDetail({ name }: WorkspaceDetailProps) {
  const [detail, setDetail] = useState<WorkspaceDetailType | null>(null)
  const [tab, setTab] = useState<Tab>('services')
  const { settings } = useSettings()
  const { show: showToast } = useToast()
  const [buildResult, setBuildResult] = useState<BuildCheckResult | null>(null)
  const [showRebuildDialog, setShowRebuildDialog] = useState(false)
  const [pendingStart, setPendingStart] = useState<{ profile: string } | null>(null)
  const [starting, setStarting] = useState(false)
  const [discoverDiff, setDiscoverDiff] = useState<DiscoverDiff | null>(null)
  const [showDiscoverDialog, setShowDiscoverDialog] = useState(false)
  const [rescanning, setRescanning] = useState(false)

  const refresh = useCallback(() => {
    window.go.api.WorkspaceAPI.GetWorkspace(name).then(setDetail).catch(console.error)
  }, [name])

  useEffect(() => {
    refresh()
    const off = window.runtime.EventsOn('service:status', () => refresh())
    return off
  }, [name, refresh])

  const handleStart = async (profile: string) => {
    setStarting(true)
    try {
      const result = await window.go.api.WorkspaceAPI.CheckBuilds(name)
      setBuildResult(result)

      if (result.hasStale) {
        if (settings.autoRebuild) {
          await window.go.api.WorkspaceAPI.StartWorkspace(name, profile, true)
          refresh()
        } else {
          setPendingStart({ profile })
          setShowRebuildDialog(true)
        }
      } else {
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

  const handleRescan = async () => {
    setRescanning(true)
    try {
      const diff = await window.go.api.WorkspaceAPI.DiscoverWorkspace(name)
      if (!diff.hasChanges) {
        showToast('No changes detected', 'info')
      } else {
        setDiscoverDiff(diff)
        setShowDiscoverDialog(true)
      }
    } catch (e) {
      console.error('Rescan failed:', e)
      showToast('Failed to scan: ' + e, 'error')
    } finally {
      setRescanning(false)
    }
  }

  const handleApplyDiscovery = async (newServices: string[], removedServices: string[]) => {
    setShowDiscoverDialog(false)
    try {
      await window.go.api.WorkspaceAPI.ApplyDiscovery(name, newServices, removedServices)
      showToast('Changes applied', 'success')
      refresh()
    } catch (e) {
      console.error('Apply discovery failed:', e)
      showToast('Failed to apply changes: ' + e, 'error')
    }
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
          <button
            onClick={handleRescan}
            disabled={rescanning}
            className="text-[11px] text-rook-muted hover:text-rook-text border border-rook-border px-2 py-1 rounded disabled:opacity-50"
          >
            {rescanning ? 'Scanning...' : 'Re-scan'}
          </button>
          <ProfileSwitcher profiles={detail.profiles} active={detail.activeProfile}
            onChange={(p) => handleStart(p)} />
          {hasRunning ? (
            <button onClick={() => window.go.api.WorkspaceAPI.StopWorkspace(name).then(refresh)}
              className="bg-rook-error text-white px-3 py-1 rounded text-[11px]">Stop All</button>
          ) : (
            <button
              onClick={() => handleStart(activeProfile)}
              disabled={starting}
              className="bg-rook-active text-black px-3 py-1 rounded text-[11px] font-semibold disabled:opacity-50"
            >
              {starting ? 'Starting...' : 'Start'}
            </button>
          )}
        </div>
      </div>
      <div className="flex border-b border-rook-border">
        {(['services', 'logs', 'environment', 'builds', 'manifest'] as Tab[]).map(t => (
          <button key={t} onClick={() => setTab(t)}
            className={`px-4 py-2 text-[11px] border-b-2 capitalize ${tab === t ? 'text-rook-text border-rook-active font-semibold' : 'text-rook-muted border-transparent'}`}>
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
        {tab === 'manifest' && <ManifestEditor workspaceName={name} />}
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

      <DiscoverDiffDialog
        open={showDiscoverDialog}
        diff={discoverDiff}
        onApply={handleApplyDiscovery}
        onCancel={() => setShowDiscoverDialog(false)}
      />
    </div>
  )
}
