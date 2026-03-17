import { createContext, useContext, useEffect, useState, useCallback, ReactNode } from 'react'

export interface Settings {
  autoRebuild: boolean
}

interface SettingsContextType {
  settings: Settings
  loading: boolean
  save: (settings: Settings) => Promise<void>
  refresh: () => Promise<void>
}

const SettingsContext = createContext<SettingsContextType | null>(null)

interface SettingsProviderProps {
  children: ReactNode
}

export function SettingsProvider({ children }: SettingsProviderProps) {
  const [settings, setSettings] = useState<Settings>({ autoRebuild: true })
  const [loading, setLoading] = useState(true)

  const refresh = useCallback(async () => {
    try {
      const s = await window.go.api.WorkspaceAPI.GetSettings()
      setSettings(s || { autoRebuild: true })
    } catch (e) {
      console.error('Failed to get settings:', e)
    } finally {
      setLoading(false)
    }
  }, [])

  const save = useCallback(async (s: Settings) => {
    await window.go.api.WorkspaceAPI.SaveSettings(s)
    setSettings(s)
  }, [])

  useEffect(() => {
    refresh()
  }, [refresh])

  return (
    <SettingsContext.Provider value={{ settings, loading, save, refresh }}>
      {children}
    </SettingsContext.Provider>
  )
}

export function useSettings(): SettingsContextType {
  const context = useContext(SettingsContext)
  if (!context) {
    throw new Error('useSettings must be used within a SettingsProvider')
  }
  return context
}
