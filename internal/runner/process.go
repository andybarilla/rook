package runner

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/andybarilla/rook/internal/workspace"
)

type processEntry struct {
	cmd    *exec.Cmd
	cancel context.CancelFunc
	output *bytes.Buffer
	mu     sync.Mutex
	done   chan struct{}
	err    error
}

type ProcessRunner struct {
	mu      sync.Mutex
	entries map[string]*processEntry
}

func NewProcessRunner() *ProcessRunner {
	return &ProcessRunner{entries: make(map[string]*processEntry)}
}

func (r *ProcessRunner) Start(ctx context.Context, name string, svc workspace.Service, ports PortMap, workDir string) (RunHandle, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	procCtx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(procCtx, "sh", "-c", svc.Command)
	cmd.Dir = workDir

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	cmd.Env = os.Environ()
	for k, v := range svc.Environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	entry := &processEntry{
		cmd:    cmd,
		cancel: cancel,
		output: &output,
		done:   make(chan struct{}),
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return RunHandle{}, fmt.Errorf("starting %s: %w", name, err)
	}

	go func() {
		entry.err = cmd.Wait()
		close(entry.done)
	}()

	r.entries[name] = entry
	return RunHandle{ID: name, Type: "process"}, nil
}

func (r *ProcessRunner) Stop(handle RunHandle) error {
	r.mu.Lock()
	entry, ok := r.entries[handle.ID]
	r.mu.Unlock()
	if !ok {
		return nil
	}
	entry.cancel()
	<-entry.done
	return nil
}

func (r *ProcessRunner) Status(handle RunHandle) (ServiceStatus, error) {
	r.mu.Lock()
	entry, ok := r.entries[handle.ID]
	r.mu.Unlock()
	if !ok {
		return StatusStopped, nil
	}
	select {
	case <-entry.done:
		if entry.err != nil {
			return StatusCrashed, nil
		}
		return StatusStopped, nil
	default:
		return StatusRunning, nil
	}
}

func (r *ProcessRunner) Logs(handle RunHandle) (io.ReadCloser, error) {
	r.mu.Lock()
	entry, ok := r.entries[handle.ID]
	r.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("no logs for %s", handle.ID)
	}
	entry.mu.Lock()
	data := make([]byte, entry.output.Len())
	copy(data, entry.output.Bytes())
	entry.mu.Unlock()
	return io.NopCloser(bytes.NewReader(data)), nil
}

// StreamLogs returns a streaming reader that polls the process output buffer.
func (r *ProcessRunner) StreamLogs(handle RunHandle) (io.ReadCloser, error) {
	r.mu.Lock()
	entry, ok := r.entries[handle.ID]
	r.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("no process for %s", handle.ID)
	}
	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		lastLen := 0
		for {
			entry.mu.Lock()
			data := entry.output.Bytes()
			currentLen := len(data)
			entry.mu.Unlock()
			if currentLen > lastLen {
				pw.Write(data[lastLen:currentLen])
				lastLen = currentLen
			}
			select {
			case <-entry.done:
				entry.mu.Lock()
				data = entry.output.Bytes()
				entry.mu.Unlock()
				if len(data) > lastLen {
					pw.Write(data[lastLen:])
				}
				return
			default:
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()
	return pr, nil
}
