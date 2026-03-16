package envgen_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/andybarilla/rook/internal/envgen"
	"github.com/andybarilla/rook/internal/workspace"
)

func TestResolveTemplates_PortAndHost(t *testing.T) {
	env := map[string]string{"DATABASE_URL": "postgres://u:p@{{.Host.postgres}}:{{.Port.postgres}}/db"}
	portMap := map[string]int{"postgres": 12345}
	result, err := envgen.ResolveTemplates(env, portMap, false)
	if err != nil {
		t.Fatal(err)
	}
	if result["DATABASE_URL"] != "postgres://u:p@localhost:12345/db" {
		t.Errorf("got %s", result["DATABASE_URL"])
	}
}

func TestResolveTemplates_DevcontainerContext(t *testing.T) {
	env := map[string]string{"REDIS_URL": "redis://{{.Host.redis}}:{{.Port.redis}}"}
	svc := workspace.Service{Image: "redis:7", Ports: []int{6379}}
	portMap := map[string]int{"redis": 12346}
	result, _ := envgen.ResolveTemplatesWithServices(env, portMap, true, map[string]workspace.Service{"redis": svc})
	if !strings.Contains(result["REDIS_URL"], "redis") {
		t.Errorf("should use service name as host: %s", result["REDIS_URL"])
	}
	if !strings.Contains(result["REDIS_URL"], "6379") {
		t.Errorf("should use internal port: %s", result["REDIS_URL"])
	}
}

func TestWriteEnvFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	envgen.WriteEnvFile(path, map[string]string{"DB_HOST": "localhost", "DB_PORT": "12345"})
	data, _ := os.ReadFile(path)
	content := string(data)
	if !strings.Contains(content, "DB_HOST=localhost") {
		t.Errorf("missing DB_HOST")
	}
	if !strings.Contains(content, "DB_PORT=12345") {
		t.Errorf("missing DB_PORT")
	}
}

func TestResolveTemplates_NoTemplates(t *testing.T) {
	result, _ := envgen.ResolveTemplates(map[string]string{"STATIC": "value"}, nil, false)
	if result["STATIC"] != "value" {
		t.Errorf("got %s", result["STATIC"])
	}
}

func TestExpandShellVars_Default(t *testing.T) {
	result := envgen.ExpandShellVars("${ROOK_TEST_UNSET_VAR:-kern}")
	if result != "kern" {
		t.Errorf("expected 'kern', got '%s'", result)
	}
}

func TestExpandShellVars_EnvSet(t *testing.T) {
	t.Setenv("ROOK_TEST_SET_VAR", "fromenv")
	result := envgen.ExpandShellVars("${ROOK_TEST_SET_VAR:-default}")
	if result != "fromenv" {
		t.Errorf("expected 'fromenv', got '%s'", result)
	}
}

func TestExpandShellVars_NoDefault(t *testing.T) {
	result := envgen.ExpandShellVars("${ROOK_TEST_UNSET_VAR}")
	if result != "" {
		t.Errorf("expected empty, got '%s'", result)
	}
}

func TestExpandShellVars_InContext(t *testing.T) {
	result := envgen.ExpandShellVars("postgres://${ROOK_TEST_UNSET_USER:-kern}:${ROOK_TEST_UNSET_PASS:-kern}@localhost:5432")
	expected := "postgres://kern:kern@localhost:5432"
	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}
