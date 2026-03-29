import { useCallback, useEffect, useState } from 'react'
import { useToast } from '../hooks/useToast'

interface ManifestEditorProps { workspaceName: string }

interface ServiceState {
  Image: string
  Command: string
  Path: string
  WorkingDir: string
  Ports: number[]
  PinPort: number
  Environment: Record<string, string>
  DependsOn: string[]
  Healthcheck: string
  Volumes: string[]
  EnvFile: string
  Build: string
  Dockerfile: string
  BuildFrom: string
}

interface ManifestState {
  Name: string
  Type: string
  Root: string
  Services: Record<string, ServiceState>
  Groups: Record<string, string[]>
  Profiles: Record<string, string[]>
}

type Mode = 'form' | 'yaml'

function emptyService(): ServiceState {
  return {
    Image: '', Command: '', Path: '', WorkingDir: '',
    Ports: [], PinPort: 0, Environment: {}, DependsOn: [],
    Healthcheck: '', Volumes: [], EnvFile: '',
    Build: '', Dockerfile: '', BuildFrom: '',
  }
}

function toManifestState(raw: any): ManifestState {
  const services: Record<string, ServiceState> = {}
  if (raw.Services) {
    for (const [name, svc] of Object.entries(raw.Services)) {
      const s = svc as any
      services[name] = {
        Image: s.Image || '',
        Command: s.Command || '',
        Path: s.Path || '',
        WorkingDir: s.WorkingDir || '',
        Ports: s.Ports || [],
        PinPort: s.PinPort || 0,
        Environment: s.Environment || {},
        DependsOn: s.DependsOn || [],
        Healthcheck: typeof s.Healthcheck === 'string' ? s.Healthcheck : (s.Healthcheck ? JSON.stringify(s.Healthcheck) : ''),
        Volumes: s.Volumes || [],
        EnvFile: s.EnvFile || '',
        Build: s.Build || '',
        Dockerfile: s.Dockerfile || '',
        BuildFrom: s.BuildFrom || '',
      }
    }
  }
  return {
    Name: raw.Name || '',
    Type: raw.Type || 'single',
    Root: raw.Root || '',
    Services: services,
    Groups: raw.Groups || {},
    Profiles: raw.Profiles || {},
  }
}

function toApiManifest(state: ManifestState): any {
  const services: Record<string, any> = {}
  for (const [name, svc] of Object.entries(state.Services)) {
    const out: any = {}
    if (svc.Image) out.Image = svc.Image
    if (svc.Command) out.Command = svc.Command
    if (svc.Path) out.Path = svc.Path
    if (svc.WorkingDir) out.WorkingDir = svc.WorkingDir
    if (svc.Ports.length) out.Ports = svc.Ports
    if (svc.PinPort) out.PinPort = svc.PinPort
    if (Object.keys(svc.Environment).length) out.Environment = svc.Environment
    if (svc.DependsOn.length) out.DependsOn = svc.DependsOn
    if (svc.Healthcheck) out.Healthcheck = svc.Healthcheck
    if (svc.Volumes.length) out.Volumes = svc.Volumes
    if (svc.EnvFile) out.EnvFile = svc.EnvFile
    if (svc.Build) out.Build = svc.Build
    if (svc.Dockerfile) out.Dockerfile = svc.Dockerfile
    if (svc.BuildFrom) out.BuildFrom = svc.BuildFrom
    services[name] = out
  }
  const manifest: any = { Name: state.Name, Type: state.Type, Services: services }
  if (state.Root) manifest.Root = state.Root
  if (Object.keys(state.Groups).length) manifest.Groups = state.Groups
  if (Object.keys(state.Profiles).length) manifest.Profiles = state.Profiles
  return manifest
}

function serviceTypeLabel(svc: ServiceState): string {
  if (svc.BuildFrom) return 'shared build'
  if (svc.Build) return 'build'
  if (svc.Image) return 'container'
  if (svc.Command) return 'process'
  return 'unknown'
}

const inputClass = 'w-full bg-rook-input border border-rook-border px-2 py-1 text-xs text-rook-text focus:outline-none focus:border-rook-active'
const labelClass = 'text-[10px] text-rook-muted uppercase tracking-wider'
const btnSecondary = 'px-2 py-1 text-[10px] bg-rook-bg border border-rook-border text-rook-text-secondary hover:bg-rook-border/50'
const btnDanger = 'px-2 py-1 text-[10px] bg-rook-bg border border-rook-error/30 text-rook-error hover:bg-rook-error/10'
const btnPrimary = 'px-3 py-1.5 text-[11px] bg-rook-active hover:bg-rook-active-hover text-white disabled:opacity-50'

export function ManifestEditor({ workspaceName }: ManifestEditorProps) {
  const [manifest, setManifest] = useState<ManifestState | null>(null)
  const [original, setOriginal] = useState<string>('')
  const [mode, setMode] = useState<Mode>('form')
  const [yamlText, setYamlText] = useState('')
  const [yamlError, setYamlError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [expandedServices, setExpandedServices] = useState<Set<string>>(new Set())
  const [error, setError] = useState<string | null>(null)
  const { show: showToast } = useToast()

  const load = useCallback(async () => {
    try {
      const raw = await window.go.api.WorkspaceAPI.GetManifest(workspaceName)
      const state = toManifestState(raw)
      setManifest(state)
      setOriginal(JSON.stringify(state))
      setError(null)
    } catch (e) {
      setError(String(e))
    }
  }, [workspaceName])

  useEffect(() => { load() }, [load])

  const dirty = manifest !== null && JSON.stringify(manifest) !== original

  const switchToYaml = async () => {
    if (!manifest || mode === 'yaml') return
    try {
      const yaml = await window.go.api.WorkspaceAPI.PreviewManifest(toApiManifest(manifest))
      setYamlText(yaml)
      setYamlError(null)
      setMode('yaml')
    } catch (e) {
      showToast('Failed to serialize: ' + e, 'error')
    }
  }

  const switchToForm = () => {
    if (mode === 'form') return
    // YAML edits are not parsed back into form state — save first to apply YAML changes
    setMode('form')
  }

  const handleSave = async () => {
    if (!manifest) return
    setSaving(true)
    try {
      if (mode === 'yaml') {
        // Save YAML by writing the text through SaveManifestYAML (not yet available),
        // so for now we save form state which was last synced when switching to YAML mode
        await window.go.api.WorkspaceAPI.SaveManifest(workspaceName, toApiManifest(manifest))
      } else {
        await window.go.api.WorkspaceAPI.SaveManifest(workspaceName, toApiManifest(manifest))
      }
      showToast('Manifest saved', 'success')
      await load()
    } catch (e) {
      showToast('Save failed: ' + e, 'error')
    } finally {
      setSaving(false)
    }
  }

  const handleRevert = () => {
    if (original) {
      setManifest(JSON.parse(original))
    }
  }

  const updateManifest = (patch: Partial<ManifestState>) => {
    setManifest(prev => prev ? { ...prev, ...patch } : prev)
  }

  const updateService = (name: string, patch: Partial<ServiceState>) => {
    setManifest(prev => {
      if (!prev) return prev
      return {
        ...prev,
        Services: { ...prev.Services, [name]: { ...prev.Services[name], ...patch } },
      }
    })
  }

  const addService = () => {
    const name = prompt('Service name:')
    if (!name || !manifest) return
    if (manifest.Services[name]) {
      showToast(`Service "${name}" already exists`, 'error')
      return
    }
    setManifest(prev => prev ? {
      ...prev,
      Services: { ...prev.Services, [name]: emptyService() },
    } : prev)
    setExpandedServices(prev => new Set([...prev, name]))
  }

  const removeService = (name: string) => {
    if (!confirm(`Remove service "${name}"?`)) return
    setManifest(prev => {
      if (!prev) return prev
      const { [name]: _, ...rest } = prev.Services
      return { ...prev, Services: rest }
    })
  }

  const renameService = (oldName: string) => {
    const newName = prompt('New name:', oldName)
    if (!newName || newName === oldName || !manifest) return
    if (manifest.Services[newName]) {
      showToast(`Service "${newName}" already exists`, 'error')
      return
    }
    setManifest(prev => {
      if (!prev) return prev
      const { [oldName]: svc, ...rest } = prev.Services
      return { ...prev, Services: { ...rest, [newName]: svc } }
    })
    setExpandedServices(prev => {
      const next = new Set(prev)
      next.delete(oldName)
      next.add(newName)
      return next
    })
  }

  const toggleService = (name: string) => {
    setExpandedServices(prev => {
      const next = new Set(prev)
      if (next.has(name)) next.delete(name)
      else next.add(name)
      return next
    })
  }

  if (error) return <div className="p-4 text-rook-error text-sm">{error}</div>
  if (!manifest) return <div className="p-4 text-rook-muted text-sm">Loading...</div>

  const serviceNames = Object.keys(manifest.Services)

  return (
    <div className="p-4 space-y-4">
      {/* Header bar */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <button
            onClick={switchToForm}
            className={`px-2 py-1 text-[10px] border ${mode === 'form' ? 'border-rook-active text-rook-active' : 'border-rook-border text-rook-muted hover:text-rook-text-secondary'}`}
          >
            Form
          </button>
          <button
            onClick={switchToYaml}
            className={`px-2 py-1 text-[10px] border ${mode === 'yaml' ? 'border-rook-active text-rook-active' : 'border-rook-border text-rook-muted hover:text-rook-text-secondary'}`}
          >
            YAML
          </button>
        </div>
        <div className="flex items-center gap-2">
          {dirty && (
            <button onClick={handleRevert} className={btnSecondary}>Revert</button>
          )}
          <button onClick={handleSave} disabled={!dirty || saving} className={btnPrimary}>
            {saving ? 'Saving...' : 'Save'}
          </button>
        </div>
      </div>

      {dirty && (
        <div className="text-[10px] text-rook-attention">Unsaved changes</div>
      )}

      {mode === 'yaml' ? (
        <YamlEditor value={yamlText} onChange={setYamlText} error={yamlError} />
      ) : (
        <FormEditor
          manifest={manifest}
          serviceNames={serviceNames}
          expandedServices={expandedServices}
          onUpdateManifest={updateManifest}
          onUpdateService={updateService}
          onAddService={addService}
          onRemoveService={removeService}
          onRenameService={renameService}
          onToggleService={toggleService}
        />
      )}
    </div>
  )
}

// --- YAML Mode ---

function YamlEditor({ value, onChange, error }: {
  value: string
  onChange: (v: string) => void
  error: string | null
}) {
  return (
    <div className="space-y-2">
      {error && <div className="text-xs text-rook-error">{error}</div>}
      <textarea
        value={value}
        onChange={e => onChange(e.target.value)}
        className="w-full h-[calc(100vh-220px)] bg-rook-input border border-rook-border p-3 text-xs font-mono text-rook-text focus:outline-none focus:border-rook-active resize-none"
        spellCheck={false}
      />
    </div>
  )
}

// --- Form Mode ---

function FormEditor({ manifest, serviceNames, expandedServices, onUpdateManifest, onUpdateService, onAddService, onRemoveService, onRenameService, onToggleService }: {
  manifest: ManifestState
  serviceNames: string[]
  expandedServices: Set<string>
  onUpdateManifest: (patch: Partial<ManifestState>) => void
  onUpdateService: (name: string, patch: Partial<ServiceState>) => void
  onAddService: () => void
  onRemoveService: (name: string) => void
  onRenameService: (name: string) => void
  onToggleService: (name: string) => void
}) {
  return (
    <div className="space-y-4">
      <Section title="workspace">
        <div className="grid grid-cols-3 gap-3">
          <Field label="name">
            <div className="text-xs text-rook-text-secondary px-2 py-1">{manifest.Name}</div>
          </Field>
          <Field label="type">
            <select
              value={manifest.Type}
              onChange={e => onUpdateManifest({ Type: e.target.value })}
              className={inputClass}
            >
              <option value="single">single</option>
              <option value="multi">multi</option>
            </select>
          </Field>
          <Field label="root">
            <input
              type="text"
              value={manifest.Root}
              onChange={e => onUpdateManifest({ Root: e.target.value })}
              className={inputClass}
              placeholder="(default)"
            />
          </Field>
        </div>
      </Section>

      <Section title="services" action={<button onClick={onAddService} className={btnSecondary}>+ Add</button>}>
        {serviceNames.length === 0 ? (
          <div className="text-xs text-rook-muted py-2">No services defined</div>
        ) : (
          <div className="space-y-1">
            {serviceNames.map(name => (
              <ServiceEditor
                key={name}
                name={name}
                service={manifest.Services[name]}
                allServiceNames={serviceNames}
                expanded={expandedServices.has(name)}
                onToggle={() => onToggleService(name)}
                onUpdate={patch => onUpdateService(name, patch)}
                onRemove={() => onRemoveService(name)}
                onRename={() => onRenameService(name)}
              />
            ))}
          </div>
        )}
      </Section>

      <MapListEditor
        title="groups"
        data={manifest.Groups}
        options={serviceNames}
        onChange={groups => onUpdateManifest({ Groups: groups })}
      />

      <MapListEditor
        title="profiles"
        data={manifest.Profiles}
        options={[...serviceNames, ...Object.keys(manifest.Groups), '*']}
        onChange={profiles => onUpdateManifest({ Profiles: profiles })}
      />
    </div>
  )
}

function Section({ title, children, action }: {
  title: string
  children: React.ReactNode
  action?: React.ReactNode
}) {
  return (
    <div>
      <div className="flex items-center justify-between mb-2">
        <h3 className="text-xs uppercase tracking-wider text-rook-text-secondary">{title}</h3>
        {action}
      </div>
      <div className="bg-rook-card border border-rook-border p-3">
        {children}
      </div>
    </div>
  )
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <label className={labelClass}>{label}</label>
      {children}
    </div>
  )
}

// --- Service Accordion ---

function ServiceEditor({ name, service, allServiceNames, expanded, onToggle, onUpdate, onRemove, onRename }: {
  name: string
  service: ServiceState
  allServiceNames: string[]
  expanded: boolean
  onToggle: () => void
  onUpdate: (patch: Partial<ServiceState>) => void
  onRemove: () => void
  onRename: () => void
}) {
  const type = serviceTypeLabel(service)

  return (
    <div className="border border-rook-border">
      <button
        onClick={onToggle}
        className="w-full flex items-center justify-between px-3 py-2 hover:bg-rook-border/20"
      >
        <div className="flex items-center gap-2">
          <span className="text-[10px] text-rook-muted">{expanded ? '\u25BC' : '\u25B6'}</span>
          <span className="text-xs text-rook-text font-medium">{name}</span>
          <span className="text-[10px] text-rook-muted px-1.5 py-0.5 border border-rook-border">{type}</span>
        </div>
      </button>

      {expanded && (
        <div className="px-3 pb-3 space-y-3 border-t border-rook-border">
          <div className="grid grid-cols-2 gap-3 mt-3">
            <Field label="image">
              <input type="text" value={service.Image} onChange={e => onUpdate({ Image: e.target.value })} className={inputClass} placeholder="e.g. postgres:16" />
            </Field>
            <Field label="command">
              <input type="text" value={service.Command} onChange={e => onUpdate({ Command: e.target.value })} className={inputClass} placeholder="e.g. npm start" />
            </Field>
            <Field label="build">
              <input type="text" value={service.Build} onChange={e => onUpdate({ Build: e.target.value })} className={inputClass} placeholder="e.g. ." />
            </Field>
            <Field label="dockerfile">
              <input type="text" value={service.Dockerfile} onChange={e => onUpdate({ Dockerfile: e.target.value })} className={inputClass} placeholder="e.g. Dockerfile.dev" />
            </Field>
            <Field label="build_from">
              <select value={service.BuildFrom} onChange={e => onUpdate({ BuildFrom: e.target.value })} className={inputClass}>
                <option value="">---</option>
                {allServiceNames.filter(n => n !== name).map(n => (
                  <option key={n} value={n}>{n}</option>
                ))}
              </select>
            </Field>
            <Field label="path">
              <input type="text" value={service.Path} onChange={e => onUpdate({ Path: e.target.value })} className={inputClass} />
            </Field>
            <Field label="working_dir">
              <input type="text" value={service.WorkingDir} onChange={e => onUpdate({ WorkingDir: e.target.value })} className={inputClass} />
            </Field>
            <Field label="env_file">
              <input type="text" value={service.EnvFile} onChange={e => onUpdate({ EnvFile: e.target.value })} className={inputClass} placeholder=".env" />
            </Field>
          </div>

          <PortsEditor ports={service.Ports} pinPort={service.PinPort} onChange={(ports, pin) => onUpdate({ Ports: ports, PinPort: pin })} />

          <KeyValueEditor label="environment" data={service.Environment} onChange={env => onUpdate({ Environment: env })} />

          <Field label="depends_on">
            <MultiSelect
              selected={service.DependsOn}
              options={allServiceNames.filter(n => n !== name)}
              onChange={deps => onUpdate({ DependsOn: deps })}
            />
          </Field>

          <StringListEditor label="volumes" items={service.Volumes} onChange={v => onUpdate({ Volumes: v })} placeholder="e.g. data:/var/lib/data" />

          <Field label="healthcheck">
            <input type="text" value={service.Healthcheck} onChange={e => onUpdate({ Healthcheck: e.target.value })} className={inputClass} placeholder="e.g. pg_isready -U postgres" />
          </Field>

          <div className="flex gap-2 pt-2 border-t border-rook-border">
            <button onClick={onRename} className={btnSecondary}>Rename</button>
            <button onClick={onRemove} className={btnDanger}>Remove</button>
          </div>
        </div>
      )}
    </div>
  )
}

// --- Ports Editor ---

function PortsEditor({ ports, pinPort, onChange }: {
  ports: number[]
  pinPort: number
  onChange: (ports: number[], pin: number) => void
}) {
  const addPort = () => onChange([...ports, 0], pinPort)
  const removePort = (i: number) => {
    const next = ports.filter((_, idx) => idx !== i)
    const newPin = ports[i] === pinPort ? 0 : pinPort
    onChange(next, newPin)
  }
  const updatePort = (i: number, val: number) => {
    const next = [...ports]
    next[i] = val
    onChange(next, pinPort)
  }

  return (
    <div>
      <div className="flex items-center justify-between">
        <label className={labelClass}>ports</label>
        <button onClick={addPort} className={btnSecondary}>+ Add</button>
      </div>
      {ports.map((p, i) => (
        <div key={i} className="flex items-center gap-2 mt-1">
          <input
            type="number"
            value={p || ''}
            onChange={e => updatePort(i, parseInt(e.target.value) || 0)}
            className={inputClass + ' w-24'}
          />
          <label className="flex items-center gap-1 text-[10px] text-rook-muted cursor-pointer">
            <input
              type="radio"
              name={`pinPort-${i}`}
              checked={pinPort === p && p > 0}
              onChange={() => onChange(ports, p)}
            />
            pin
          </label>
          <button onClick={() => removePort(i)} className="text-[10px] text-rook-error hover:text-rook-error/80">&times;</button>
        </div>
      ))}
    </div>
  )
}

// --- Key-Value Editor ---

function KeyValueEditor({ label, data, onChange }: {
  label: string
  data: Record<string, string>
  onChange: (data: Record<string, string>) => void
}) {
  const entries = Object.entries(data)

  const addEntry = () => {
    const key = prompt('Variable name:')
    if (!key) return
    onChange({ ...data, [key]: '' })
  }

  const removeEntry = (key: string) => {
    const { [key]: _, ...rest } = data
    onChange(rest)
  }

  const updateValue = (key: string, value: string) => {
    onChange({ ...data, [key]: value })
  }

  return (
    <div>
      <div className="flex items-center justify-between">
        <label className={labelClass}>{label}</label>
        <button onClick={addEntry} className={btnSecondary}>+ Add</button>
      </div>
      {entries.map(([key, val]) => (
        <div key={key} className="flex items-center gap-2 mt-1">
          <span className="text-[10px] text-rook-text-secondary font-mono w-32 shrink-0 truncate" title={key}>{key}</span>
          <input
            type="text"
            value={val}
            onChange={e => updateValue(key, e.target.value)}
            className={inputClass + ' flex-1'}
          />
          <button onClick={() => removeEntry(key)} className="text-[10px] text-rook-error hover:text-rook-error/80">&times;</button>
        </div>
      ))}
    </div>
  )
}

// --- Multi-Select ---

function MultiSelect({ selected, options, onChange }: {
  selected: string[]
  options: string[]
  onChange: (selected: string[]) => void
}) {
  const toggle = (opt: string) => {
    if (selected.includes(opt)) {
      onChange(selected.filter(s => s !== opt))
    } else {
      onChange([...selected, opt])
    }
  }

  if (options.length === 0) return <div className="text-[10px] text-rook-muted py-1">No options</div>

  return (
    <div className="flex flex-wrap gap-2 mt-1">
      {options.map(opt => (
        <label key={opt} className="flex items-center gap-1 text-[10px] text-rook-text-secondary cursor-pointer">
          <input type="checkbox" checked={selected.includes(opt)} onChange={() => toggle(opt)} className="border-rook-border" />
          {opt}
        </label>
      ))}
    </div>
  )
}

// --- String List Editor ---

function StringListEditor({ label, items, onChange, placeholder }: {
  label: string
  items: string[]
  onChange: (items: string[]) => void
  placeholder?: string
}) {
  const addItem = () => onChange([...items, ''])
  const removeItem = (i: number) => onChange(items.filter((_, idx) => idx !== i))
  const updateItem = (i: number, val: string) => {
    const next = [...items]
    next[i] = val
    onChange(next)
  }

  return (
    <div>
      <div className="flex items-center justify-between">
        <label className={labelClass}>{label}</label>
        <button onClick={addItem} className={btnSecondary}>+ Add</button>
      </div>
      {items.map((item, i) => (
        <div key={i} className="flex items-center gap-2 mt-1">
          <input
            type="text"
            value={item}
            onChange={e => updateItem(i, e.target.value)}
            className={inputClass + ' flex-1'}
            placeholder={placeholder}
          />
          <button onClick={() => removeItem(i)} className="text-[10px] text-rook-error hover:text-rook-error/80">&times;</button>
        </div>
      ))}
    </div>
  )
}

// --- Groups/Profiles Editor ---

function MapListEditor({ title, data, options, onChange }: {
  title: string
  data: Record<string, string[]>
  options: string[]
  onChange: (data: Record<string, string[]>) => void
}) {
  const entries = Object.entries(data)

  const addEntry = () => {
    const key = prompt(`${title.slice(0, -1)} name:`)
    if (!key) return
    onChange({ ...data, [key]: [] })
  }

  const removeEntry = (key: string) => {
    const { [key]: _, ...rest } = data
    onChange(rest)
  }

  const updateEntry = (key: string, values: string[]) => {
    onChange({ ...data, [key]: values })
  }

  return (
    <Section title={title} action={<button onClick={addEntry} className={btnSecondary}>+ Add</button>}>
      {entries.length === 0 ? (
        <div className="text-xs text-rook-muted py-1">No {title} defined</div>
      ) : (
        <div className="space-y-3">
          {entries.map(([key, values]) => (
            <div key={key}>
              <div className="flex items-center justify-between mb-1">
                <span className="text-xs text-rook-text font-medium">{key}</span>
                <button onClick={() => removeEntry(key)} className={btnDanger}>&times;</button>
              </div>
              <MultiSelect selected={values} options={options} onChange={v => updateEntry(key, v)} />
            </div>
          ))}
        </div>
      )}
    </Section>
  )
}
