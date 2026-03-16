package workspace_test

import (
	"testing"

	"github.com/andybarilla/rook/internal/workspace"
)

func TestServiceIsContainer(t *testing.T) {
	svc := workspace.Service{Image: "postgres:16-alpine"}
	if !svc.IsContainer() {
		t.Error("service with image should be a container")
	}
	if svc.IsProcess() {
		t.Error("service with image should not be a process")
	}
}

func TestServiceIsProcess(t *testing.T) {
	svc := workspace.Service{Command: "air"}
	if !svc.IsProcess() {
		t.Error("service with command should be a process")
	}
	if svc.IsContainer() {
		t.Error("service with command should not be a container")
	}
}

func TestServiceIsBuildContainer(t *testing.T) {
	svc := workspace.Service{Build: "."}
	if !svc.IsContainer() {
		t.Error("service with build should be a container")
	}
	if svc.IsProcess() {
		t.Error("service with build should not be a process")
	}
}

func TestServiceBuildWithCommand(t *testing.T) {
	svc := workspace.Service{Build: ".", Command: "./server -worker"}
	if !svc.IsContainer() {
		t.Error("service with build+command should be a container")
	}
	if svc.IsProcess() {
		t.Error("service with build+command should not be a process")
	}
}

func TestWorkspaceServiceNames(t *testing.T) {
	ws := workspace.Workspace{
		Name: "test",
		Services: map[string]workspace.Service{
			"postgres": {Image: "postgres:16"},
			"app":      {Command: "air"},
		},
	}
	names := ws.ServiceNames()
	if len(names) != 2 {
		t.Fatalf("expected 2 services, got %d", len(names))
	}
}
