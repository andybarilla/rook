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

func TestService_IsContainer_WithBuildFrom(t *testing.T) {
	svc := workspace.Service{BuildFrom: "server"}
	if !svc.IsContainer() {
		t.Error("service with BuildFrom should be a container")
	}
}

func TestService_IsProcess_WithBuildFrom(t *testing.T) {
	svc := workspace.Service{BuildFrom: "server", Command: "run.sh"}
	if svc.IsProcess() {
		t.Error("service with BuildFrom should not be a process even with command")
	}
}

func TestManifest_Validate_BuildFromValid(t *testing.T) {
	m := &workspace.Manifest{
		Services: map[string]workspace.Service{
			"server": {Build: ".", Ports: []int{8080}},
			"worker": {BuildFrom: "server", Command: "work"},
		},
	}
	if err := m.Validate(); err != nil {
		t.Errorf("valid build_from should not error: %v", err)
	}
}

func TestManifest_Validate_BuildFromMutuallyExclusiveWithBuild(t *testing.T) {
	m := &workspace.Manifest{
		Services: map[string]workspace.Service{
			"server": {Build: "."},
			"worker": {BuildFrom: "server", Build: "."},
		},
	}
	if err := m.Validate(); err == nil {
		t.Error("build_from with build should error")
	}
}

func TestManifest_Validate_BuildFromMutuallyExclusiveWithImage(t *testing.T) {
	m := &workspace.Manifest{
		Services: map[string]workspace.Service{
			"server": {Build: "."},
			"worker": {BuildFrom: "server", Image: "nginx"},
		},
	}
	if err := m.Validate(); err == nil {
		t.Error("build_from with image should error")
	}
}

func TestManifest_Validate_BuildFromTargetMissing(t *testing.T) {
	m := &workspace.Manifest{
		Services: map[string]workspace.Service{
			"worker": {BuildFrom: "nonexistent"},
		},
	}
	if err := m.Validate(); err == nil {
		t.Error("build_from referencing missing service should error")
	}
}

func TestManifest_Validate_BuildFromTargetHasNoBuild(t *testing.T) {
	m := &workspace.Manifest{
		Services: map[string]workspace.Service{
			"server": {Image: "nginx"},
			"worker": {BuildFrom: "server"},
		},
	}
	if err := m.Validate(); err == nil {
		t.Error("build_from referencing service without build should error")
	}
}

func TestManifest_Validate_BuildFromChainDisallowed(t *testing.T) {
	m := &workspace.Manifest{
		Services: map[string]workspace.Service{
			"server":  {Build: "."},
			"worker":  {BuildFrom: "server"},
			"worker2": {BuildFrom: "worker"},
		},
	}
	if err := m.Validate(); err == nil {
		t.Error("chained build_from should error")
	}
}

func TestManifest_Validate_NoBuildFrom(t *testing.T) {
	m := &workspace.Manifest{
		Services: map[string]workspace.Service{
			"web": {Image: "nginx", Ports: []int{80}},
		},
	}
	if err := m.Validate(); err != nil {
		t.Errorf("manifest without build_from should validate: %v", err)
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
