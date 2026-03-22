package cli

import (
	"os"
	"path/filepath"
	"strings"
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

func TestResolveAndLoadWorkspace_AutoInit(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgDir)

	wsDir := t.TempDir()
	manifest := []byte("name: autows\nservices:\n  api:\n    image: node\n    ports:\n      - 3000\n")
	os.WriteFile(filepath.Join(wsDir, "rook.yaml"), manifest, 0644)

	origDir, _ := os.Getwd()
	os.Chdir(wsDir)
	defer os.Chdir(origDir)

	cctx, _ := newCLIContext()

	// Simulate user typing "y\n" on stdin
	r, w, _ := os.Pipe()
	w.WriteString("y\n")
	w.Close()

	ws, err := cctx.resolveAndLoadWorkspace(nil, r)
	if err != nil {
		t.Fatal(err)
	}
	if ws.Name != "autows" {
		t.Errorf("expected workspace name autows, got %s", ws.Name)
	}

	// Verify it was registered
	_, err = cctx.registry.Get("autows")
	if err != nil {
		t.Error("workspace was not registered after auto-init")
	}
}

func TestResolveAndLoadWorkspace_Declined(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgDir)

	wsDir := t.TempDir()
	manifest := []byte("name: declinews\nservices:\n  api:\n    image: node\n    ports:\n      - 3000\n")
	os.WriteFile(filepath.Join(wsDir, "rook.yaml"), manifest, 0644)

	origDir, _ := os.Getwd()
	os.Chdir(wsDir)
	defer os.Chdir(origDir)

	cctx, _ := newCLIContext()

	r, w, _ := os.Pipe()
	w.WriteString("n\n")
	w.Close()

	_, err := cctx.resolveAndLoadWorkspace(nil, r)
	if err == nil {
		t.Fatal("expected error when user declines")
	}
	if !strings.Contains(err.Error(), "rook init") {
		t.Errorf("expected hint about rook init, got: %s", err.Error())
	}
}

func TestResolveAndLoadWorkspace_ExplicitNameNotFound(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgDir)

	cctx, _ := newCLIContext()

	r, w, _ := os.Pipe()
	w.Close()

	_, err := cctx.resolveAndLoadWorkspace([]string{"nonexistent"}, r)
	if err == nil {
		t.Fatal("expected error for unregistered explicit name")
	}
	if strings.Contains(err.Error(), "Initialize") {
		t.Error("should not prompt for explicit name arg")
	}
}

func TestResolveAndLoadWorkspace_AlreadyRegistered(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgDir)

	wsDir := t.TempDir()
	manifest := []byte("name: existingws\nservices:\n  db:\n    image: postgres\n    ports:\n      - 5432\n")
	os.WriteFile(filepath.Join(wsDir, "rook.yaml"), manifest, 0644)

	cctx, _ := newCLIContext()
	cctx.registry.Register("existingws", wsDir)

	origDir, _ := os.Getwd()
	os.Chdir(wsDir)
	defer os.Chdir(origDir)

	r, w, _ := os.Pipe()
	w.Close()

	ws, err := cctx.resolveAndLoadWorkspace(nil, r)
	if err != nil {
		t.Fatal(err)
	}
	if ws.Name != "existingws" {
		t.Errorf("expected existingws, got %s", ws.Name)
	}
}

func TestResolveAndLoadWorkspace_NonTTY(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgDir)

	wsDir := t.TempDir()
	manifest := []byte("name: nonttyws\nservices:\n  api:\n    image: node\n    ports:\n      - 3000\n")
	os.WriteFile(filepath.Join(wsDir, "rook.yaml"), manifest, 0644)

	origDir, _ := os.Getwd()
	os.Chdir(wsDir)
	defer os.Chdir(origDir)

	cctx, _ := newCLIContext()

	// Closed pipe simulates non-interactive stdin (EOF immediately)
	r, w, _ := os.Pipe()
	w.Close()

	_, err := cctx.resolveAndLoadWorkspace(nil, r)
	if err == nil {
		t.Fatal("expected error for non-TTY stdin")
	}
	if !strings.Contains(err.Error(), "rook init") {
		t.Errorf("expected hint about rook init, got: %s", err.Error())
	}
}
