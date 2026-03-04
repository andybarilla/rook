package databases

type ServiceType string

const (
	MySQL    ServiceType = "mysql"
	Postgres ServiceType = "postgres"
	Redis    ServiceType = "redis"
)

// AllServiceTypes lists every supported database in display order.
var AllServiceTypes = []ServiceType{MySQL, Postgres, Redis}

type ServiceConfig struct {
	Port    int
	DataDir string
}

type ServiceStatus int

const (
	StatusStopped ServiceStatus = iota
	StatusRunning
)

// ServiceInfo is the per-service detail exposed to Core / GUI.
type ServiceInfo struct {
	Type      ServiceType `json:"type"`
	Enabled   bool        `json:"enabled"`
	Running   bool        `json:"running"`
	Autostart bool        `json:"autostart"`
	Port      int         `json:"port"`
}

// DBRunner abstracts database process management.
type DBRunner interface {
	Start(svc ServiceType, cfg ServiceConfig) error
	Stop(svc ServiceType) error
	Status(svc ServiceType) ServiceStatus
}
