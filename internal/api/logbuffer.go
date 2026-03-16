package api

import (
	"sync"
	"time"
)

// LogLine represents a single log line from a workspace service.
type LogLine struct {
	Workspace string    `json:"workspace"`
	Service   string    `json:"service"`
	Line      string    `json:"line"`
	Timestamp time.Time `json:"timestamp"`
}

// LogBuffer is a rolling per-workspace log buffer that stores log lines
// up to a configurable maximum per workspace.
type LogBuffer struct {
	mu       sync.Mutex
	maxLines int
	lines    map[string][]LogLine
}

// NewLogBuffer creates a new LogBuffer that retains at most maxLines per workspace.
func NewLogBuffer(maxLines int) *LogBuffer {
	return &LogBuffer{
		maxLines: maxLines,
		lines:    make(map[string][]LogLine),
	}
}

// Add appends a log line for the given workspace and service, trimming old
// entries if the buffer exceeds maxLines. Returns the created LogLine.
func (b *LogBuffer) Add(workspace, service, line string) LogLine {
	entry := LogLine{
		Workspace: workspace,
		Service:   service,
		Line:      line,
		Timestamp: time.Now(),
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.lines[workspace] = append(b.lines[workspace], entry)
	if len(b.lines[workspace]) > b.maxLines {
		b.lines[workspace] = b.lines[workspace][len(b.lines[workspace])-b.maxLines:]
	}

	return entry
}

// Get returns the last limit log lines for the given workspace. If service is
// non-empty, only lines for that service are returned. If limit is 0 or
// negative, all matching lines are returned.
func (b *LogBuffer) Get(workspace, service string, limit int) []LogLine {
	b.mu.Lock()
	defer b.mu.Unlock()

	all := b.lines[workspace]
	if all == nil {
		return nil
	}

	var filtered []LogLine
	if service == "" {
		filtered = make([]LogLine, len(all))
		copy(filtered, all)
	} else {
		for _, l := range all {
			if l.Service == service {
				filtered = append(filtered, l)
			}
		}
	}

	if limit > 0 && len(filtered) > limit {
		filtered = filtered[len(filtered)-limit:]
	}

	return filtered
}
