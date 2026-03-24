import { useEffect, useRef, useState, useCallback } from 'react'
import type { LogLine, LogEvent } from '../hooks/useWails'

interface LogViewerProps {
  workspaceName: string
  services: string[]
}

const SERVICE_COLORS = ['text-rook-active', 'text-rook-attention', 'text-rook-text', 'text-rook-text-secondary', 'text-rook-active', 'text-rook-attention']

export function LogViewer({ workspaceName, services }: LogViewerProps) {
  const [activeTab, setActiveTab] = useState<string>('')
  const [lines, setLines] = useState<LogLine[]>([])
  const [filter, setFilter] = useState('')
  const [autoScroll, setAutoScroll] = useState(true)
  const scrollRef = useRef<HTMLDivElement>(null)

  const colorMap = useCallback((service: string) => {
    const idx = services.indexOf(service) % SERVICE_COLORS.length
    return SERVICE_COLORS[idx >= 0 ? idx : 0]
  }, [services])

  useEffect(() => {
    const service = activeTab || ''
    window.go.api.WorkspaceAPI.GetLogs(workspaceName, service, 500)
      .then(l => setLines(l || []))
      .catch(console.error)
  }, [workspaceName, activeTab])

  useEffect(() => {
    const off = window.runtime.EventsOn('service:log', (event: LogEvent) => {
      if (event.workspace !== workspaceName) return
      if (activeTab && event.service !== activeTab) return
      const line: LogLine = { workspace: event.workspace, service: event.service, line: event.line, timestamp: event.timestamp }
      setLines(prev => [...prev.slice(-999), line])
    })
    return off
  }, [workspaceName, activeTab])

  useEffect(() => {
    if (autoScroll && scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    }
  }, [lines, autoScroll])

  const handleScroll = () => {
    if (!scrollRef.current) return
    const { scrollTop, scrollHeight, clientHeight } = scrollRef.current
    setAutoScroll(scrollHeight - scrollTop - clientHeight < 50)
  }

  const filteredLines = filter ? lines.filter(l => l.line.toLowerCase().includes(filter.toLowerCase())) : lines

  return (
    <div className="flex flex-col h-full relative">
      <div className="flex border-b border-rook-border px-4">
        <TabButton label="All" active={activeTab === ''} onClick={() => setActiveTab('')} />
        {services.map(s => (
          <TabButton key={s} label={s} active={activeTab === s} onClick={() => setActiveTab(s)} className={colorMap(s)} />
        ))}
      </div>
      <div className="px-4 py-1.5 border-b border-rook-border">
        <input type="text" placeholder="Filter logs..." value={filter} onChange={e => setFilter(e.target.value)}
          className="w-full bg-rook-input text-rook-text-secondary border border-rook-border px-2 py-1 text-xs" />
      </div>
      <div ref={scrollRef} onScroll={handleScroll} className="flex-1 overflow-auto px-4 py-2 font-mono text-[9px] leading-relaxed bg-rook-input">
        {filteredLines.map((l, i) => (
          <div key={`${l.timestamp}-${l.service}-${i}`}>
            {activeTab === '' && <span className={colorMap(l.service)}>[{l.service.padEnd(12)}]</span>}
            {' '}<span className="text-rook-text-secondary">{l.line}</span>
          </div>
        ))}
      </div>
      {!autoScroll && (
        <button onClick={() => { setAutoScroll(true); scrollRef.current?.scrollTo({ top: scrollRef.current.scrollHeight }) }}
          className="absolute bottom-4 right-4 bg-rook-active text-white text-xs px-3 py-1">
          Jump to bottom
        </button>
      )}
    </div>
  )
}

function TabButton({ label, active, onClick, className = '' }: { label: string; active: boolean; onClick: () => void; className?: string }) {
  return (
    <button onClick={onClick} className={`px-2.5 py-1.5 text-[10px] border-b-2 ${active ? 'text-rook-text border-rook-active font-semibold' : `${className || 'text-rook-muted'} border-transparent`}`}>
      {label}
    </button>
  )
}
