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
