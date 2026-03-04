# flock-databases Plugin Design

**Date:** 2026-03-04
**Status:** Approved

## Overview

The `flock-databases` plugin manages local database services (MySQL, PostgreSQL, Redis) as a single unified `ServicePlugin`. It starts, stops, and monitors database processes using binaries found on the system PATH.

## Decisions

- **Process management**: Plugin manages actual database processes (start/stop/status)
- **Binary source**: System PATH — binaries must be pre-installed; plugin manages lifecycle only
- **Scope**: Global services — one instance per database type, shared across all sites
- **Services**: MySQL, PostgreSQL, Redis (all three in initial implementation)
- **Versions**: Single version per type (whatever is on PATH)
- **Autostart**: Configurable per service (default off)
- **GUI**: Backend plugin + basic services panel in UI

## Plugin Architecture

### Unified Plugin (Approach 1)

One `flock-databases` plugin implementing `ServicePlugin`, with an internal `DBRunner` interface abstracting the three database types. Follows the same pattern as `flock-php` with `FPMRunner`.

### DBRunner Interface

```go
type ServiceType string

const (
    MySQL    ServiceType = "mysql"
    Postgres ServiceType = "postgres"
    Redis    ServiceType = "redis"
)

type DBRunner interface {
    Start(svc ServiceType, cfg ServiceConfig) error
    Stop(svc ServiceType) error
    Status(svc ServiceType) ServiceStatus
}

type ServiceConfig struct {
    Port    int
    DataDir string
}

type ServiceStatus int

const (
    StatusStopped ServiceStatus = iota
    StatusRunning
)
```

### Plugin Struct

```go
type Plugin struct {
    runner   DBRunner
    host     plugin.Host
    config   *Config
    services map[ServiceType]bool // which services are currently running
}
```

**Lifecycle:**
- `Init()` — loads config from `~/.config/flock/databases.json`, checks PATH for binaries
- `Start()` — starts all services marked as autostart (if enabled)
- `Stop()` — stops all running services
- `ServiceStatus()` — returns Running if any service is up, Stopped if none

**Per-service control** (exposed via Core/Wails bindings):
- `StartSvc(svc ServiceType) error`
- `StopSvc(svc ServiceType) error`
- `ServiceStatuses() []ServiceInfo`

## Configuration

**File:** `~/.config/flock/databases.json`

```json
{
  "mysql": {
    "enabled": true,
    "autostart": false,
    "port": 3306,
    "dataDir": ""
  },
  "postgres": {
    "enabled": true,
    "autostart": false,
    "port": 5432,
    "dataDir": ""
  },
  "redis": {
    "enabled": true,
    "autostart": true,
    "port": 6379,
    "dataDir": ""
  }
}
```

- **`enabled`**: Whether the binary was found on PATH (auto-detected)
- **`autostart`**: Start when Flock starts (user-configured)
- **`port`**: Configurable to avoid conflicts with existing installations
- **`dataDir`**: Defaults to `~/.local/share/flock/databases/{mysql,postgres,redis}/`

Missing config file → created with defaults. Missing binary → `enabled: false`, logged.

## Process Management

### MySQL

- **Binary check**: `mysqld` on PATH
- **Init (first run)**: `mysqld --initialize-insecure --datadir=<dir>`
- **Start**: `mysqld --datadir=<dir> --port=<port> --socket=<dir>/mysql.sock --pid-file=<dir>/mysql.pid`
- **Stop**: `mysqladmin --socket=<dir>/mysql.sock shutdown`
- **Status**: PID file + process alive check

### PostgreSQL

- **Binary check**: `pg_ctl` on PATH
- **Init (first run)**: `initdb -D <dir>`
- **Start**: `pg_ctl -D <dir> -l <dir>/postgres.log -o "-p <port>" start`
- **Stop**: `pg_ctl -D <dir> stop`
- **Status**: `pg_ctl -D <dir> status`

### Redis

- **Binary check**: `redis-server` on PATH
- **Init**: None needed (Redis creates data files on start)
- **Start**: `redis-server --port <port> --dir <dir> --daemonize yes --pidfile <dir>/redis.pid`
- **Stop**: `redis-cli -p <port> shutdown`
- **Status**: PID file + process alive check

All commands via `os/exec`. Processes run as background daemons. Data directories auto-initialized on first start.

## Core Integration

### Config Changes

```go
type Config struct {
    // ... existing fields
    DBRunner databases.DBRunner
}
```

### Core Changes

```go
type Core struct {
    // ... existing fields
    dbPlugin *databases.Plugin
}
```

Registration in `NewCore()`:
```go
dbPlugin := databases.NewPlugin(cfg.DBRunner)
pluginMgr.Register(dbPlugin)
```

### Wails Bindings

```go
func (a *App) DatabaseServices() []databases.ServiceInfo
func (a *App) StartDatabase(svc string) error
func (a *App) StopDatabase(svc string) error
```

### ServiceInfo (returned to frontend)

```go
type ServiceInfo struct {
    Type      ServiceType
    Enabled   bool   // binary found on PATH
    Running   bool   // currently running
    Autostart bool   // start with Flock
    Port      int    // configured port
}
```

## GUI: Services Panel

New "Services" section below existing "Sites" section in `App.svelte`.

### ServiceList.svelte

Displays a row for each database service:
- Service name (MySQL, PostgreSQL, Redis)
- Status indicator (green = running, gray = stopped, red = unavailable)
- Port number
- Start/Stop toggle button
- Disabled state if binary not found ("not installed" message)

Follows existing dark theme and styling patterns. No autostart toggle in GUI — config file only for now.

## Error Handling

- Binary not found → `enabled: false`, logged, not an error
- Process start failure → logged, service stays stopped, others unaffected
- Process stop failure → logged, best-effort (don't block Flock shutdown)
- Data dir init failure → error from Start, service stays stopped
- Config file missing → create with defaults
- Config file corrupt → error from Init, plugin degraded

## File Layout

```
internal/databases/
├── databases.go       # Plugin struct, ServicePlugin implementation
├── databases_test.go  # Plugin tests with mock DBRunner
├── runner.go          # DBRunner interface, ServiceType, ServiceConfig
├── process.go         # ProcessRunner (concrete os/exec implementation)
├── process_test.go    # ProcessRunner tests
└── config.go          # Config struct, load/save JSON
```

## Testing

- `DBRunner` interface enables full mock testing (no real databases)
- Mock `DBRunner` in test file (same pattern as `mockFPMRunner`)
- ~15-20 test cases: plugin lifecycle, per-service start/stop, status aggregation, config loading, autostart, error cases
- TDD: tests written before implementation
