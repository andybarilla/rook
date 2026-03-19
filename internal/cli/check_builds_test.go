package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/andybarilla/rook/internal/buildcache"
	"github.com/andybarilla/rook/internal/workspace"
)

func TestCheckBuildsCmd_UsesCachePath(t *testing.T) {
	wsRoot := "/tmp/testws"
	expectedPath := filepath.Join(wsRoot, ".rook", ".cache", "build-cache.json")
	actualPath := buildCachePath(wsRoot)
	if actualPath != expectedPath {
		t.Errorf("expected %s, got %s", expectedPath, actualPath)
	}
}

func TestCheckBuildsCmd_Help(t *testing.T) {
	// Verify command is registered and has help
	cmd := NewCheckBuildsCmd()
	if cmd == nil {
		t.Fatal("expected non-nil command")
	}
	if cmd.Use != "check-builds [workspace]" {
		t.Errorf("Use: got %q", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("expected non-empty Short")
	}
}

func TestPrintCheckBuildsText_BuildFrom(t *testing.T) {
	// Test that build_from services are displayed correctly
	results := map[string]buildcache.StaleResult{
		"api":    {},
		"client": {},
	}
	services := map[string]workspace.Service{
		"api":    {Build: "api"},
		"client": {BuildFrom: "api"},
	}

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := printCheckBuildsText(results, services, false)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var output bytes.Buffer
	output.ReadFrom(r)
	text := output.String()

	// Check that build_from service shows "uses image from"
	if !bytes.Contains([]byte(text), []byte("client: uses image from api")) {
		t.Errorf("expected build_from service to show 'uses image from api', got: %s", text)
	}
}

func TestPrintCheckBuildsJSON_BuildFrom(t *testing.T) {
	// Test that build_from services are displayed correctly in JSON
	results := map[string]buildcache.StaleResult{
		"api":    {},
		"client": {},
	}
	services := map[string]workspace.Service{
		"api":    {Build: "api"},
		"client": {BuildFrom: "api"},
	}

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := printCheckBuildsJSON(results, services, false)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var output bytes.Buffer
	output.ReadFrom(r)

	var result checkBuildsJSONOutput
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	// Check that client has build_from status with reason pointing to api
	if status, ok := result.Services["client"]; !ok {
		t.Fatal("expected client service in output")
	} else if status.Status != "build_from" {
		t.Errorf("expected status 'build_from', got %q", status.Status)
	} else if status.Reason != "api" {
		t.Errorf("expected reason 'api', got %q", status.Reason)
	}
}
