package runner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// PIDInfo is the data stored in a PID file for a running process service.
type PIDInfo struct {
	PID       int       `json:"pid"`
	Command   string    `json:"command"`
	StartedAt time.Time `json:"started_at"`
}

// PIDDirPath returns the directory where PID files are stored for a workspace.
func PIDDirPath(wsRoot string) string {
	return filepath.Join(wsRoot, ".rook", ".cache", "pids")
}

// WritePIDFile writes a PID file for the named service into dir.
func WritePIDFile(dir, serviceName string, info PIDInfo) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating pid dir: %w", err)
	}
	data, err := json.Marshal(info)
	if err != nil {
		return err
	}
	return os.WriteFile(pidFilePath(dir, serviceName), data, 0644)
}

// ReadPIDFile reads the PID file for the named service from dir.
func ReadPIDFile(dir, serviceName string) (*PIDInfo, error) {
	data, err := os.ReadFile(pidFilePath(dir, serviceName))
	if err != nil {
		return nil, err
	}
	var info PIDInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// RemovePIDFile removes the PID file for the named service. No error if missing.
func RemovePIDFile(dir, serviceName string) error {
	err := os.Remove(pidFilePath(dir, serviceName))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// ListPIDFiles returns service names that have PID files in dir.
// Returns empty slice (not error) if dir does not exist.
func ListPIDFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".pid") {
			names = append(names, strings.TrimSuffix(name, ".pid"))
		}
	}
	return names, nil
}

// IsProcessAlive checks whether a process with the given PID is still running.
// Uses signal 0 on Unix (Linux/macOS). Returns false for invalid PIDs.
// Note: on Linux, signal 0 succeeds for zombie processes (exited but not yet
// reaped). This is acceptable — stale PID files are cleaned up on next check.
func IsProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}

func pidFilePath(dir, serviceName string) string {
	return filepath.Join(dir, serviceName+".pid")
}
