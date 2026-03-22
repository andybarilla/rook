package cli

import (
	"bytes"
	"testing"

	"github.com/andybarilla/rook/internal/workspace"
)

func TestFormatRebuildPrompt_SingleServiceNoConsumers(t *testing.T) {
	var buf bytes.Buffer
	stale := map[string][]string{"api": {"Dockerfile modified"}}
	services := map[string]workspace.Service{
		"api": {Build: "."},
	}
	formatRebuildPrompt(&buf, stale, services, nil)

	got := buf.String()
	want := "1 service(s) need rebuild:\n  - api (Dockerfile modified)\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatRebuildPrompt_WithSingleConsumer(t *testing.T) {
	var buf bytes.Buffer
	stale := map[string][]string{"api": {"Dockerfile modified"}}
	services := map[string]workspace.Service{
		"api":    {Build: "."},
		"worker": {BuildFrom: "api"},
	}
	resolved := []string{"api", "worker"}
	formatRebuildPrompt(&buf, stale, services, resolved)

	got := buf.String()
	want := "1 service(s) need rebuild:\n  - api (Dockerfile modified)\n    also used by: worker\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatRebuildPrompt_WithMultipleConsumers(t *testing.T) {
	var buf bytes.Buffer
	stale := map[string][]string{"api": {"context files changed"}}
	services := map[string]workspace.Service{
		"api":       {Build: "."},
		"worker":    {BuildFrom: "api"},
		"scheduler": {BuildFrom: "api"},
	}
	resolved := []string{"api", "worker", "scheduler"}
	formatRebuildPrompt(&buf, stale, services, resolved)

	got := buf.String()
	want := "1 service(s) need rebuild:\n  - api (context files changed)\n    also used by: scheduler, worker\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatRebuildPrompt_ConsumerNotInProfile(t *testing.T) {
	var buf bytes.Buffer
	stale := map[string][]string{"api": {"Dockerfile modified"}}
	services := map[string]workspace.Service{
		"api":    {Build: "."},
		"worker": {BuildFrom: "api"},
	}
	resolved := []string{"api"} // worker not in profile
	formatRebuildPrompt(&buf, stale, services, resolved)

	got := buf.String()
	want := "1 service(s) need rebuild:\n  - api (Dockerfile modified)\n"
	if got != want {
		t.Errorf("consumer not in profile should be excluded, got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatRebuildPrompt_NoReason(t *testing.T) {
	var buf bytes.Buffer
	stale := map[string][]string{"api": {}}
	services := map[string]workspace.Service{
		"api": {Build: "."},
	}
	formatRebuildPrompt(&buf, stale, services, nil)

	got := buf.String()
	want := "1 service(s) need rebuild:\n  - api\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatRebuildPrompt_MultipleStaleSources(t *testing.T) {
	var buf bytes.Buffer
	stale := map[string][]string{
		"api":      {"Dockerfile modified"},
		"frontend": {"context files changed"},
	}
	services := map[string]workspace.Service{
		"api":      {Build: "."},
		"worker":   {BuildFrom: "api"},
		"frontend": {Build: "./web"},
		"ssr":      {BuildFrom: "frontend"},
	}
	resolved := []string{"api", "worker", "frontend", "ssr"}
	formatRebuildPrompt(&buf, stale, services, resolved)

	got := buf.String()
	want := "2 service(s) need rebuild:\n  - api (Dockerfile modified)\n    also used by: worker\n  - frontend (context files changed)\n    also used by: ssr\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestBuildFromConsumers_ReturnsConsumersOfRebuiltSources(t *testing.T) {
	services := map[string]workspace.Service{
		"api":       {Build: ".", ForceBuild: true},
		"worker":    {BuildFrom: "api"},
		"scheduler": {BuildFrom: "api"},
		"frontend":  {Build: "./web"},
	}
	resolved := []string{"api", "worker", "scheduler", "frontend"}

	got := buildFromConsumers(services, resolved)

	if len(got) != 2 {
		t.Fatalf("expected 2 consumers, got %d: %v", len(got), got)
	}
	if got[0] != "scheduler" || got[1] != "worker" {
		t.Errorf("expected [scheduler worker], got %v", got)
	}
}

func TestBuildFromConsumers_ExcludesNonProfileConsumers(t *testing.T) {
	services := map[string]workspace.Service{
		"api":    {Build: ".", ForceBuild: true},
		"worker": {BuildFrom: "api"},
	}
	resolved := []string{"api"}

	got := buildFromConsumers(services, resolved)

	if len(got) != 0 {
		t.Errorf("expected 0 consumers (worker not in profile), got %v", got)
	}
}

func TestBuildFromConsumers_IgnoresNonRebuiltSources(t *testing.T) {
	services := map[string]workspace.Service{
		"api":    {Build: "."},
		"worker": {BuildFrom: "api"},
	}
	resolved := []string{"api", "worker"}

	got := buildFromConsumers(services, resolved)

	if len(got) != 0 {
		t.Errorf("expected 0 consumers (source not rebuilt), got %v", got)
	}
}
