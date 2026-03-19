import { useEffect, useState, useCallback } from 'react'

export interface WorkspaceInfo {
  name: string
  path: string
  serviceCount: number
  runningCount: number
  activeProfile?: string
}

export interface ServiceInfo {
  name: string
  image?: string
  command?: string
  status: 'starting' | 'running' | 'stopped' | 'crashed'
  port?: number
  dependsOn?: string[]
  hasBuild: boolean
  buildStatus?: 'up_to_date' | 'needs_rebuild' | 'no_build_context'
}

export interface WorkspaceDetail {
  name: string
  path: string
  services: ServiceInfo[]
  profiles: string[]
  groups?: Record<string, string[]>
  activeProfile?: string
}

export interface PortEntry {
  workspace: string
  service: string
  port: number
  pinned?: boolean
}

export interface LogLine {
  workspace: string
  service: string
  line: string
  timestamp: number
}

export interface StatusEvent {
  workspace: string
  service: string
  status: string
  port?: number
}

export interface LogEvent {
  workspace: string
  service: string
  line: string
  timestamp: number
}

// New types for settings and builds
export interface Settings {
  autoRebuild: boolean
}

export interface BuildStatus {
  name: string
  hasBuild: boolean
  status: 'up_to_date' | 'needs_rebuild' | 'no_build_context'
  reasons?: string[]
}

export interface BuildCheckResult {
  services: BuildStatus[]
  hasStale: boolean
}

export interface ServiceDiff {
  name: string
  image?: string
  build?: string
  reason?: string
}

export interface DiscoverDiff {
  source: string
  newServices: ServiceDiff[]
  removedServices: ServiceDiff[]
  hasChanges: boolean
}

// Wails runtime globals
declare global {
  interface Window {
    go: {
      api: {
        WorkspaceAPI: {
          ListWorkspaces(): Promise<WorkspaceInfo[]>
          GetWorkspace(name: string): Promise<WorkspaceDetail>
          AddWorkspace(path: string): Promise<any>
          RemoveWorkspace(name: string): Promise<void>
          StartWorkspace(name: string, profile: string, forceBuild: boolean): Promise<void>
          StopWorkspace(name: string): Promise<void>
          StartService(workspace: string, service: string): Promise<void>
          StopService(workspace: string, service: string): Promise<void>
          RestartService(workspace: string, service: string): Promise<void>
          GetPorts(): Promise<PortEntry[]>
          GetEnv(workspace: string): Promise<Record<string, any[]>>
          GetLogs(workspace: string, service: string, lines: number): Promise<LogLine[]>
          PreviewManifest(manifest: any): Promise<string>
          SaveManifest(name: string, manifest: any): Promise<void>
          GetSettings(): Promise<Settings>
          SaveSettings(settings: Settings): Promise<void>
          CheckBuilds(workspace: string): Promise<BuildCheckResult>
          ResetPorts(): Promise<void>
          DiscoverWorkspace(name: string): Promise<DiscoverDiff>
          ApplyDiscovery(name: string, newServices: string[], removedServices: string[]): Promise<void>
        }
      }
    }
    runtime: {
      EventsOn(event: string, callback: (...data: any) => void): () => void
      EventsEmit(event: string, ...data: any): void
    }
  }
}

export function useWorkspaces() {
  const [workspaces, setWorkspaces] = useState<WorkspaceInfo[]>([])
  const [loading, setLoading] = useState(true)

  const refresh = useCallback(async () => {
    try {
      const list = await window.go.api.WorkspaceAPI.ListWorkspaces()
      setWorkspaces(list || [])
    } catch (e) {
      console.error('Failed to list workspaces:', e)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    refresh()
    const off1 = window.runtime.EventsOn('workspace:changed', () => refresh())
    const off2 = window.runtime.EventsOn('service:status', () => refresh())
    return () => { off1(); off2() }
  }, [refresh])

  return { workspaces, loading, refresh }
}

export function usePorts() {
  const [ports, setPorts] = useState<PortEntry[]>([])
  useEffect(() => {
    window.go.api.WorkspaceAPI.GetPorts().then(p => setPorts(p || [])).catch(console.error)
  }, [])
  return ports
}
