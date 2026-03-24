package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/andybarilla/rook/internal/workspace"
)

func setupEnvRewriteWorkspace(t *testing.T, envContent string, manifest *workspace.Manifest) (wsDir string, cfgDir string) {
	t.Helper()
	cfgDir = t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgDir)

	wsDir = t.TempDir()
	if envContent != "" {
		if err := os.WriteFile(filepath.Join(wsDir, ".env"), []byte(envContent), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := workspace.WriteManifest(filepath.Join(wsDir, "rook.yaml"), manifest); err != nil {
		t.Fatal(err)
	}

	// Register the workspace via init
	initCmd := newInitCmd()
	initCmd.SetArgs([]string{wsDir})
	if err := initCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	return wsDir, cfgDir
}

func TestEnvRewriteCmd_RewritesURLInManifest(t *testing.T) {
	wsDir, _ := setupEnvRewriteWorkspace(t,
		"DATABASE_URL=postgres://user:pass@localhost:5432/mydb\n",
		&workspace.Manifest{
			Name: "testws",
			Type: workspace.TypeSingle,
			Services: map[string]workspace.Service{
				"app": {
					Command: "node server.js",
					Ports:   []int{3000},
					EnvFile: ".env",
				},
				"postgres": {
					Image: "postgres:16",
					Ports: []int{5432},
				},
			},
		},
	)

	cmd := newEnvCmd()
	cmd.SetArgs([]string{"rewrite", "DATABASE_URL", "postgres", "testws"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	updated, err := workspace.ParseManifest(filepath.Join(wsDir, "rook.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	expected := "postgres://user:pass@{{.Host.postgres}}:{{.Port.postgres}}/mydb"
	if updated.Services["app"].Environment["DATABASE_URL"] != expected {
		t.Errorf("got %q, want %q", updated.Services["app"].Environment["DATABASE_URL"], expected)
	}
}

func TestEnvRewriteCmd_ErrorOnMissingVar(t *testing.T) {
	setupEnvRewriteWorkspace(t,
		"OTHER_VAR=something\n",
		&workspace.Manifest{
			Name: "testws",
			Type: workspace.TypeSingle,
			Services: map[string]workspace.Service{
				"app":      {Command: "node server.js", EnvFile: ".env"},
				"postgres": {Image: "postgres:16", Ports: []int{5432}},
			},
		},
	)

	cmd := newEnvCmd()
	cmd.SetArgs([]string{"rewrite", "DATABASE_URL", "postgres", "testws"})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error for missing var")
	}
}

func TestEnvRewriteCmd_ErrorOnMissingService(t *testing.T) {
	setupEnvRewriteWorkspace(t,
		"DATABASE_URL=postgres://localhost:5432/db\n",
		&workspace.Manifest{
			Name: "testws",
			Type: workspace.TypeSingle,
			Services: map[string]workspace.Service{
				"app": {Command: "node server.js", EnvFile: ".env"},
			},
		},
	)

	cmd := newEnvCmd()
	cmd.SetArgs([]string{"rewrite", "DATABASE_URL", "postgres", "testws"})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error for missing service")
	}
}

func TestEnvRewriteCmd_ErrorOnNoEnvFile(t *testing.T) {
	setupEnvRewriteWorkspace(t,
		"",
		&workspace.Manifest{
			Name: "testws",
			Type: workspace.TypeSingle,
			Services: map[string]workspace.Service{
				"app":      {Command: "node server.js"},
				"postgres": {Image: "postgres:16", Ports: []int{5432}},
			},
		},
	)

	cmd := newEnvCmd()
	cmd.SetArgs([]string{"rewrite", "DATABASE_URL", "postgres", "testws"})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error when no services have env_file")
	}
}

func TestEnvRewriteCmd_ErrorOnUnrecognizedValue(t *testing.T) {
	setupEnvRewriteWorkspace(t,
		"MY_VAR=some_opaque_string\n",
		&workspace.Manifest{
			Name: "testws",
			Type: workspace.TypeSingle,
			Services: map[string]workspace.Service{
				"app":      {Command: "node server.js", EnvFile: ".env"},
				"postgres": {Image: "postgres:16", Ports: []int{5432}},
			},
		},
	)

	cmd := newEnvCmd()
	cmd.SetArgs([]string{"rewrite", "MY_VAR", "postgres", "testws"})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error for unrecognized value")
	}
}

func TestEnvRewriteCmd_Idempotent(t *testing.T) {
	wsDir, _ := setupEnvRewriteWorkspace(t,
		"DATABASE_URL=postgres://user:pass@localhost:5432/mydb\n",
		&workspace.Manifest{
			Name: "testws",
			Type: workspace.TypeSingle,
			Services: map[string]workspace.Service{
				"app": {
					Command: "node server.js",
					Ports:   []int{3000},
					EnvFile: ".env",
				},
				"postgres": {
					Image: "postgres:16",
					Ports: []int{5432},
				},
			},
		},
	)

	// Run twice
	for i := 0; i < 2; i++ {
		cmd := newEnvCmd()
		cmd.SetArgs([]string{"rewrite", "DATABASE_URL", "postgres", "testws"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("run %d: %v", i+1, err)
		}
	}

	updated, err := workspace.ParseManifest(filepath.Join(wsDir, "rook.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	expected := "postgres://user:pass@{{.Host.postgres}}:{{.Port.postgres}}/mydb"
	if updated.Services["app"].Environment["DATABASE_URL"] != expected {
		t.Errorf("got %q, want %q", updated.Services["app"].Environment["DATABASE_URL"], expected)
	}
}
