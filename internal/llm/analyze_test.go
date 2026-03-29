package llm

import (
	"context"
	"testing"
)

type mockProvider struct {
	response string
	err      error
}

func (m *mockProvider) Complete(_ context.Context, _ Request) (Response, error) {
	if m.err != nil {
		return Response{}, m.err
	}
	return Response{Content: m.response}, nil
}

func TestAnalyzeDockerfile(t *testing.T) {
	t.Run("parses_valid_json_response", func(t *testing.T) {
		p := &mockProvider{
			response: `[{"name":"api","type":"process","command":"go run ./cmd/api","ports":[8080],"depends_on":["postgres"],"reasoning":"Go API server"},{"name":"postgres","type":"container","image":"postgres:16","ports":[5432],"reasoning":"Database"}]`,
		}

		suggestions, err := AnalyzeDockerfile(context.Background(), p, "FROM golang:1.22\nEXPOSE 8080", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(suggestions) != 2 {
			t.Fatalf("expected 2 suggestions, got %d", len(suggestions))
		}
		if suggestions[0].Name != "api" {
			t.Errorf("expected 'api', got %q", suggestions[0].Name)
		}
		if suggestions[0].Type != "process" {
			t.Errorf("expected 'process', got %q", suggestions[0].Type)
		}
		if suggestions[1].Image != "postgres:16" {
			t.Errorf("expected 'postgres:16', got %q", suggestions[1].Image)
		}
	})

	t.Run("handles_markdown_fences", func(t *testing.T) {
		p := &mockProvider{
			response: "```json\n[{\"name\":\"web\",\"type\":\"process\",\"command\":\"npm run dev\",\"reasoning\":\"Node app\"}]\n```",
		}

		suggestions, err := AnalyzeDockerfile(context.Background(), p, "FROM node:22", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(suggestions) != 1 {
			t.Fatalf("expected 1 suggestion, got %d", len(suggestions))
		}
	})

	t.Run("returns_error_for_invalid_json", func(t *testing.T) {
		p := &mockProvider{
			response: "I think you should have these services: api and postgres",
		}

		_, err := AnalyzeDockerfile(context.Background(), p, "FROM golang:1.22", "", "")
		if err == nil {
			t.Fatal("expected error for non-JSON response")
		}
	})

	t.Run("returns_error_on_provider_failure", func(t *testing.T) {
		p := &mockProvider{
			err: context.DeadlineExceeded,
		}

		_, err := AnalyzeDockerfile(context.Background(), p, "FROM golang:1.22", "", "")
		if err == nil {
			t.Fatal("expected error on provider failure")
		}
	})
}

func TestAnalyzeRepo(t *testing.T) {
	t.Run("parses_valid_response", func(t *testing.T) {
		p := &mockProvider{
			response: `[{"name":"api","type":"process","command":"go run ./cmd/api","reasoning":"Go service"}]`,
		}

		configFiles := map[string]string{
			"go.mod": "module github.com/example/app\ngo 1.22\n",
		}

		suggestions, err := AnalyzeRepo(context.Background(), p, "cmd/\n  api/\ngo.mod\n", configFiles)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(suggestions) != 1 {
			t.Fatalf("expected 1 suggestion, got %d", len(suggestions))
		}
	})

	t.Run("handles_empty_response", func(t *testing.T) {
		p := &mockProvider{response: "[]"}

		suggestions, err := AnalyzeRepo(context.Background(), p, "", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(suggestions) != 0 {
			t.Fatalf("expected 0 suggestions, got %d", len(suggestions))
		}
	})
}
