package runner

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/andybarilla/rook/internal/workspace"
)

type processEntry struct {
	cmd     *exec.Cmd
	cancel  context.CancelFunc
	output  *syncBuffer
	logFile *os.File
	done    chan struct{}
	err     error
}

type ProcessRunner struct {
	mu      sync.Mutex
	entries map[string]*processEntry
	logDir  string
	pidDir  string
}

func NewProcessRunner() *ProcessRunner {
	return &ProcessRunner{entries: make(map[string]*processEntry)}
}

// SetLogDir sets the directory for persistent process log files.
// Must be called before Start(). When set, process output is teed to
// <logDir>/<service>.log in addition to the in-memory buffer.
func (r *ProcessRunner) SetLogDir(dir string) {
	r.logDir = dir
}

// SetPIDDir sets the directory for PID files. When set, Start writes a PID
// file and Stop removes it.
func (r *ProcessRunner) SetPIDDir(dir string) {
	r.pidDir = dir
}

func (r *ProcessRunner) Start(ctx context.Context, name string, svc workspace.Service, ports PortMap, workDir string) (RunHandle, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	procCtx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(procCtx, "sh", "-c", svc.Command)
	cmd.Dir = workDir

	var output syncBuffer
	var logFile *os.File

	if r.logDir != "" {
		if err := os.MkdirAll(r.logDir, 0755); err != nil {
			cancel()
			return RunHandle{}, fmt.Errorf("creating log dir: %w", err)
		}
		logPath := filepath.Join(r.logDir, name+".log")
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			cancel()
			return RunHandle{}, fmt.Errorf("opening log file for %s: %w", name, err)
		}
		fmt.Fprintf(f, "--- rook up %s ---\n", time.Now().Format(time.RFC3339))
		logFile = f
		cmd.Stdout = io.MultiWriter(&output, f)
		cmd.Stderr = io.MultiWriter(&output, f)
	} else {
		cmd.Stdout = &output
		cmd.Stderr = &output
	}
	cmd.Env = os.Environ()
	for k, v := range svc.Environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	entry := &processEntry{
		cmd:     cmd,
		cancel:  cancel,
		output:  &output,
		logFile: logFile,
		done:    make(chan struct{}),
	}

	if err := cmd.Start(); err != nil {
		cancel()
		if logFile != nil {
			logFile.Close()
		}
		return RunHandle{}, fmt.Errorf("starting %s: %w", name, err)
	}

	go func() {
		entry.err = cmd.Wait()
		if entry.logFile != nil {
			entry.logFile.Close()
		}
		close(entry.done)
	}()

	r.entries[name] = entry

	if r.pidDir != "" {
		WritePIDFile(r.pidDir, name, PIDInfo{
			PID:       cmd.Process.Pid,
			Command:   svc.Command,
			StartedAt: time.Now(),
		})
	}

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
	if r.pidDir != "" {
		RemovePIDFile(r.pidDir, handle.ID)
	}
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
	data := entry.output.Bytes()
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
			data := entry.output.Bytes()
			if len(data) > lastLen {
				pw.Write(data[lastLen:])
				lastLen = len(data)
			}
			select {
			case <-entry.done:
				data = entry.output.Bytes()
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
