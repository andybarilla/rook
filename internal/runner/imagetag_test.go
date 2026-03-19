package runner

import (
	"testing"

	"github.com/andybarilla/rook/internal/workspace"
)

func TestResolveImageTag_BuildFrom(t *testing.T) {
	r := NewDockerRunner("rook_myapp")
	tag := r.resolveImageTag("worker", workspace.Service{BuildFrom: "server"})
	want := "rook-myapp-server:latest"
	if tag != want {
		t.Errorf("got %q, want %q", tag, want)
	}
}

func TestResolveImageTag_Build(t *testing.T) {
	r := NewDockerRunner("rook_myapp")
	tag := r.resolveImageTag("api", workspace.Service{Build: "."})
	want := "rook-myapp-api:latest"
	if tag != want {
		t.Errorf("got %q, want %q", tag, want)
	}
}

func TestResolveImageTag_Image(t *testing.T) {
	r := NewDockerRunner("rook_myapp")
	tag := r.resolveImageTag("db", workspace.Service{Image: "postgres:16"})
	want := "postgres:16"
	if tag != want {
		t.Errorf("got %q, want %q", tag, want)
	}
}
