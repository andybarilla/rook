package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewCLIContext(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	ctx, err := newCLIContext()
	if err != nil {
		t.Fatal(err)
	}
	if ctx.registry == nil {
		t.Error("expected registry")
	}
	if ctx.portAlloc == nil {
		t.Error("expected port allocator")
	}
}

func TestResolveWorkspaceName_FromArgs(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	ctx, _ := newCLIContext()
	ctx.registry.Register("myws", "/some/path")
	name, err := ctx.resolveWorkspaceName([]string{"myws"})
	if err != nil {
		t.Fatal(err)
	}
	if name != "myws" {
		t.Errorf("expected myws, got %s", name)
	}
}

func TestResolveWorkspaceName_FromCurrentDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	wsDir := t.TempDir()
	os.WriteFile(filepath.Join(wsDir, "rook.yaml"), []byte("name: localws\nservices: {}"), 0644)
	origDir, _ := os.Getwd()
	os.Chdir(wsDir)
	defer os.Chdir(origDir)
	ctx, _ := newCLIContext()
	name, err := ctx.resolveWorkspaceName(nil)
	if err != nil {
		t.Fatal(err)
	}
	if name != "localws" {
		t.Errorf("expected localws, got %s", name)
	}
}

func TestInitFromManifest(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgDir)

	// Create a workspace directory with rook.yaml
	wsDir := t.TempDir()
	manifest := []byte("name: testws\nservices:\n  web:\n    image: nginx\n    ports:\n      - 8080\n")
	os.WriteFile(filepath.Join(wsDir, "rook.yaml"), manifest, 0644)

	cctx, err := newCLIContext()
	if err != nil {
		t.Fatal(err)
	}

	err = cctx.initFromManifest(wsDir)
	if err != nil {
		t.Fatal(err)
	}

	// Verify workspace is registered
	entry, err := cctx.registry.Get("testws")
	if err != nil {
		t.Fatalf("workspace not registered: %v", err)
	}
	if entry.Path != wsDir {
		t.Errorf("expected path %s, got %s", wsDir, entry.Path)
	}

	// Verify port was allocated (Get returns LookupResult, not (int, error))
	result := cctx.portAlloc.Get("testws", "web")
	if !result.OK {
		t.Fatal("port not allocated")
	}
	if result.Port < 10000 || result.Port > 60000 {
		t.Errorf("port %d out of expected range", result.Port)
	}

	// Verify .rook/.gitignore was created
	gitignorePath := filepath.Join(wsDir, ".rook", ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		t.Error(".rook/.gitignore was not created")
	}
}
