package databases

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// BinaryFor returns the primary binary name for a service type.
func BinaryFor(svc ServiceType) string {
	switch svc {
	case MySQL:
		return "mysqld"
	case Postgres:
		return "pg_ctl"
	case Redis:
		return "redis-server"
	}
	return ""
}

// CheckBinary reports whether the named binary is on PATH.
func CheckBinary(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// ProcessRunner manages database processes using os/exec.
type ProcessRunner struct {
	procs map[ServiceType]*os.Process
}

// NewProcessRunner creates a ProcessRunner.
func NewProcessRunner() *ProcessRunner {
	return &ProcessRunner{
		procs: map[ServiceType]*os.Process{},
	}
}

func (r *ProcessRunner) Start(svc ServiceType, cfg ServiceConfig) error {
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	switch svc {
	case MySQL:
		return r.startMySQL(cfg)
	case Postgres:
		return r.startPostgres(cfg)
	case Redis:
		return r.startRedis(cfg)
	}
	return fmt.Errorf("unknown service type: %s", svc)
}

func (r *ProcessRunner) Stop(svc ServiceType) error {
	p, ok := r.procs[svc]
	if !ok {
		return nil
	}
	if err := stopProcess(p); err != nil {
		return fmt.Errorf("stop %s: %w", svc, err)
	}
	delete(r.procs, svc)
	return nil
}

func (r *ProcessRunner) Status(svc ServiceType) ServiceStatus {
	p, ok := r.procs[svc]
	if !ok {
		return StatusStopped
	}
	if !isProcessAlive(p) {
		delete(r.procs, svc)
		return StatusStopped
	}
	return StatusRunning
}

// --- MySQL ---

func (r *ProcessRunner) startMySQL(cfg ServiceConfig) error {
	entries, _ := os.ReadDir(cfg.DataDir)
	if len(entries) == 0 {
		cmd := exec.Command("mysqld", "--initialize-insecure", "--datadir="+cfg.DataDir)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("mysql init: %s: %w", string(out), err)
		}
	}

	cmd := exec.Command("mysqld",
		"--datadir="+cfg.DataDir,
		"--port="+strconv.Itoa(cfg.Port),
		"--socket="+cfg.DataDir+"/mysql.sock",
		"--pid-file="+cfg.DataDir+"/mysql.pid",
	)
	setSysProcAttr(cmd)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start mysqld: %w", err)
	}
	r.procs[MySQL] = cmd.Process
	return nil
}

// --- PostgreSQL ---

func (r *ProcessRunner) startPostgres(cfg ServiceConfig) error {
	entries, _ := os.ReadDir(cfg.DataDir)
	if len(entries) == 0 {
		cmd := exec.Command("initdb", "-D", cfg.DataDir)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("postgres init: %s: %w", string(out), err)
		}
	}

	cmd := exec.Command("pg_ctl",
		"-D", cfg.DataDir,
		"-l", cfg.DataDir+"/postgres.log",
		"-o", "-p "+strconv.Itoa(cfg.Port),
		"start",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("start postgres: %s: %w", string(out), err)
	}

	pidData, err := os.ReadFile(cfg.DataDir + "/postmaster.pid")
	if err == nil {
		lines := strings.SplitN(string(pidData), "\n", 2)
		if pid, err := strconv.Atoi(strings.TrimSpace(lines[0])); err == nil {
			if proc, err := os.FindProcess(pid); err == nil {
				r.procs[Postgres] = proc
			}
		}
	}
	return nil
}

// --- Redis ---

func (r *ProcessRunner) startRedis(cfg ServiceConfig) error {
	cmd := exec.Command("redis-server",
		"--port", strconv.Itoa(cfg.Port),
		"--dir", cfg.DataDir,
		"--daemonize", "yes",
		"--pidfile", cfg.DataDir+"/redis.pid",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("start redis: %s: %w", string(out), err)
	}

	pidData, err := os.ReadFile(cfg.DataDir + "/redis.pid")
	if err == nil {
		if pid, err := strconv.Atoi(strings.TrimSpace(string(pidData))); err == nil {
			if proc, err := os.FindProcess(pid); err == nil {
				r.procs[Redis] = proc
			}
		}
	}
	return nil
}
