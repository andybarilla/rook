package e2e_test

import (
	"strings"
	"testing"
)

func TestEnv_Basic(t *testing.T) {
	binPath := buildRook(t)

	manifest := `
name: envws
type: single
services:
  app:
    image: node
    ports: [3000]
    environment:
      NODE_ENV: development
      API_URL: http://localhost:3000
`
	wsDir, configDir := createTestWorkspace(t, manifest)

	_, err := runRook(t, binPath, configDir, "init", wsDir)
	if err != nil {
		t.Fatalf("init failed: %s", err)
	}

	output, err := runRook(t, binPath, configDir, "env", "envws")
	if err != nil {
		t.Fatalf("env failed: %s", output)
	}
	if !strings.Contains(output, "NODE_ENV") || !strings.Contains(output, "development") {
		t.Errorf("expected NODE_ENV in output, got: %s", output)
	}
	if !strings.Contains(output, "API_URL") {
		t.Errorf("expected API_URL in output, got: %s", output)
	}
}

func TestEnv_TemplateResolution(t *testing.T) {
	binPath := buildRook(t)

	manifest := `
name: templatews
type: single
services:
  web:
    image: nginx
    ports: [8080]
    environment:
      PORT: "{{.Port.web}}"
  api:
    image: node
    ports: [3000]
    environment:
      WEB_URL: "http://{{.Host.web}}:{{.Port.web}}"
`
	wsDir, configDir := createTestWorkspace(t, manifest)

	_, err := runRook(t, binPath, configDir, "init", wsDir)
	if err != nil {
		t.Fatalf("init failed: %s", err)
	}

	output, err := runRook(t, binPath, configDir, "env", "templatews")
	if err != nil {
		t.Fatalf("env failed: %s", output)
	}

	// PORT should be resolved to a number (not the template)
	if strings.Contains(output, "{{.Port.web}}") {
		t.Errorf("template should be resolved, got: %s", output)
	}
}

func TestEnv_UnknownWorkspace(t *testing.T) {
	binPath := buildRook(t)
	_, configDir := createTestWorkspace(t, "")

	_, err := runRook(t, binPath, configDir, "env", "nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown workspace")
	}
}
