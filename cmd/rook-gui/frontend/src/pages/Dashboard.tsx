import { useState } from 'react'
import { WorkspaceInfo, usePorts } from '../hooks/useWails'
import { useSettings } from '../hooks/useSettings'
import { useToast } from '../hooks/useToast'
import { ConfirmDialog } from '../components/ConfirmDialog'

interface DashboardProps {
  workspaces: WorkspaceInfo[]
}

export function Dashboard({ workspaces }: DashboardProps) {
  const ports = usePorts()
  const { settings, save: saveSettings } = useSettings()
  const { show: showToast } = useToast()
  const [showResetConfirm, setShowResetConfirm] = useState(false)
  const [resetting, setResetting] = useState(false)
  const runningCount = workspaces.reduce((sum, ws) => sum + ws.runningCount, 0)
  const totalServices = workspaces.reduce((sum, ws) => sum + ws.serviceCount, 0)
  const stoppedCount = totalServices - runningCount

  const handleResetPorts = async () => {
    setResetting(true)
    try {
      await window.go.api.WorkspaceAPI.ResetPorts()
      setShowResetConfirm(false)
      // Refresh the page to show cleared ports
      window.location.reload()
    } catch (e) {
      console.error('Reset ports failed:', e)
      alert('Failed to reset ports: ' + e)
    } finally {
      setResetting(false)
    }
  }

  return (
    <div className="p-4">
      <h1 className="text-lg font-semibold text-rook-text">Dashboard</h1>
      <p className="text-[11px] text-rook-muted mb-4">
        {workspaces.length} workspaces · {runningCount} services running
      </p>
      <div className="grid grid-cols-3 gap-2 mb-4">
        <StatCard label="Running" value={runningCount} color="text-rook-running" />
        <StatCard label="Stopped" value={stoppedCount} color="text-rook-muted" />
        <StatCard label="Ports Used" value={ports.length} color="text-rook-partial" />
      </div>
      <p className="text-[10px] uppercase tracking-wider text-rook-text-secondary mb-2">Port Allocations</p>
      <div className="bg-rook-card rounded-md text-xs overflow-hidden mb-4">
        <div className="grid grid-cols-[1fr_1fr_80px] px-2.5 py-2 text-rook-muted border-b border-rook-border">
          <span>Workspace</span><span>Service</span><span>Port</span>
        </div>
        {ports.length === 0 ? (
          <div className="px-2.5 py-3 text-rook-muted text-center">No ports allocated</div>
        ) : (
          ports.map((p) => (
            <div key={`${p.workspace}-${p.service}`} className="grid grid-cols-[1fr_1fr_80px] px-2.5 py-1.5 text-rook-text-secondary border-b border-rook-border last:border-b-0">
              <span>{p.workspace}</span>
              <span>{p.service}</span>
              <span className="font-mono">{p.port}{p.pinned && <span className="text-rook-muted ml-1">(pinned)</span>}</span>
            </div>
          ))
        )}
      </div>
      <button
        onClick={() => setShowResetConfirm(true)}
        className="text-[10px] text-rook-crashed hover:underline"
      >
        Reset Ports
      </button>

      <div className="mt-6 pt-4 border-t border-rook-border">
        <label className="flex items-center gap-2 cursor-pointer">
          <input
            type="checkbox"
            checked={settings.autoRebuild}
            onChange={async (e) => {
              try {
                await saveSettings({ ...settings, autoRebuild: e.target.checked })
              } catch (err) {
                showToast('Failed to save settings', 'error')
              }
            }}
            className="rounded border-rook-border"
          />
          <span className="text-xs text-rook-text-secondary">Auto-rebuild on stale</span>
        </label>
      </div>

      <ConfirmDialog
        open={showResetConfirm}
        title="Reset Port Allocations"
        message="This will stop all running containers and clear port allocations. Continue?"
        confirmLabel={resetting ? 'Resetting...' : 'Reset Ports'}
        variant="danger"
        onConfirm={handleResetPorts}
        onCancel={() => setShowResetConfirm(false)}
      />
    </div>
  )
}

function StatCard({ label, value, color }: { label: string; value: number; color: string }) {
  return (
    <div className="bg-rook-card rounded-md p-3 text-center">
      <div className={`text-xl font-bold ${color}`}>{value}</div>
      <div className="text-[10px] text-rook-muted">{label}</div>
    </div>
  )
}
