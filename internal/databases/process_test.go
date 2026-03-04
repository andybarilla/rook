package databases_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/andybarilla/flock/internal/databases"
)

func TestProcessRunnerStatusStoppedByDefault(t *testing.T) {
	r := databases.NewProcessRunner()
	if r.Status(databases.MySQL) != databases.StatusStopped {
		t.Error("expected StatusStopped for MySQL by default")
	}
	if r.Status(databases.Postgres) != databases.StatusStopped {
		t.Error("expected StatusStopped for Postgres by default")
	}
	if r.Status(databases.Redis) != databases.StatusStopped {
		t.Error("expected StatusStopped for Redis by default")
	}
}

func TestProcessRunnerStartCreatesDataDir(t *testing.T) {
	r := databases.NewProcessRunner()
	dataDir := filepath.Join(t.TempDir(), "mysql-data")

	// Expect Start to fail (no mysqld on PATH in test) but dataDir should be created
	_ = r.Start(databases.MySQL, databases.ServiceConfig{Port: 13306, DataDir: dataDir})

	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		t.Error("expected data dir to be created")
	}
}

func TestProcessRunnerBinaryNames(t *testing.T) {
	tests := []struct {
		svc  databases.ServiceType
		want string
	}{
		{databases.MySQL, "mysqld"},
		{databases.Postgres, "pg_ctl"},
		{databases.Redis, "redis-server"},
	}
	for _, tt := range tests {
		got := databases.BinaryFor(tt.svc)
		if got != tt.want {
			t.Errorf("BinaryFor(%s) = %q, want %q", tt.svc, got, tt.want)
		}
	}
}

func TestProcessRunnerCheckBinary(t *testing.T) {
	// A binary that definitely doesn't exist
	if databases.CheckBinary("__nonexistent_binary_12345__") {
		t.Error("expected CheckBinary to return false for nonexistent binary")
	}
	// "go" should exist in test environment
	if !databases.CheckBinary("go") {
		t.Error("expected CheckBinary to return true for 'go'")
	}
}
