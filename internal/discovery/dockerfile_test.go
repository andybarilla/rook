package discovery

import (
	"reflect"
	"testing"
)

func TestParseDockerfile(t *testing.T) {
	t.Run("expose_ports", func(t *testing.T) {
		sig := ParseDockerfile([]byte("FROM node:22\nEXPOSE 8080 3000\n"))
		if !reflect.DeepEqual(sig.ExposedPorts, []int{8080, 3000}) {
			t.Errorf("expected [8080, 3000], got %v", sig.ExposedPorts)
		}
	})

	t.Run("expose_with_protocol", func(t *testing.T) {
		sig := ParseDockerfile([]byte("FROM node:22\nEXPOSE 5432/tcp\n"))
		if !reflect.DeepEqual(sig.ExposedPorts, []int{5432}) {
			t.Errorf("expected [5432], got %v", sig.ExposedPorts)
		}
	})

	t.Run("apt_get_install", func(t *testing.T) {
		sig := ParseDockerfile([]byte("FROM ubuntu\nRUN apt-get install -y postgresql-client redis-tools curl\n"))
		if len(sig.AptPackages) != 3 {
			t.Fatalf("expected 3 packages, got %d: %v", len(sig.AptPackages), sig.AptPackages)
		}
		if sig.AptPackages[0] != "postgresql-client" {
			t.Errorf("expected postgresql-client, got %q", sig.AptPackages[0])
		}
	})

	t.Run("apk_add", func(t *testing.T) {
		sig := ParseDockerfile([]byte("FROM alpine\nRUN apk add --no-cache postgresql-client\n"))
		if len(sig.AptPackages) != 1 || sig.AptPackages[0] != "postgresql-client" {
			t.Errorf("expected [postgresql-client], got %v", sig.AptPackages)
		}
	})

	t.Run("inferred_deps", func(t *testing.T) {
		sig := ParseDockerfile([]byte("FROM ubuntu\nRUN apt-get install -y postgresql-client redis-tools\n"))
		if !reflect.DeepEqual(sig.InferredDeps, []string{"postgres", "redis"}) {
			t.Errorf("expected [postgres, redis], got %v", sig.InferredDeps)
		}
	})

	t.Run("inferred_deps_dedup", func(t *testing.T) {
		sig := ParseDockerfile([]byte("FROM ubuntu\nRUN apt-get install -y postgresql-client libpq-dev\n"))
		if !reflect.DeepEqual(sig.InferredDeps, []string{"postgres"}) {
			t.Errorf("expected [postgres] (deduped), got %v", sig.InferredDeps)
		}
	})

	t.Run("named_stages", func(t *testing.T) {
		sig := ParseDockerfile([]byte("FROM golang:1.22 AS builder\nRUN go build\nFROM alpine\nCOPY --from=builder /app /app\n"))
		if !reflect.DeepEqual(sig.Stages, []string{"builder"}) {
			t.Errorf("expected [builder], got %v", sig.Stages)
		}
	})

	t.Run("cmd_json_array", func(t *testing.T) {
		sig := ParseDockerfile([]byte("FROM golang\nCMD [\"go\", \"run\", \"./cmd/api\"]\n"))
		if sig.EntryCmd != "go run ./cmd/api" {
			t.Errorf("expected 'go run ./cmd/api', got %q", sig.EntryCmd)
		}
	})

	t.Run("cmd_shell_form", func(t *testing.T) {
		sig := ParseDockerfile([]byte("FROM golang\nCMD go run ./cmd/api\n"))
		if sig.EntryCmd != "go run ./cmd/api" {
			t.Errorf("expected 'go run ./cmd/api', got %q", sig.EntryCmd)
		}
	})

	t.Run("entrypoint_wins_over_cmd", func(t *testing.T) {
		sig := ParseDockerfile([]byte("FROM alpine\nCMD [\"go\", \"run\", \".\"]\nENTRYPOINT [\"/app/start.sh\"]\n"))
		if sig.EntryCmd != "/app/start.sh" {
			t.Errorf("expected '/app/start.sh', got %q", sig.EntryCmd)
		}
	})

	t.Run("minimal_dockerfile", func(t *testing.T) {
		sig := ParseDockerfile([]byte("FROM alpine\nRUN echo hello\n"))
		if len(sig.ExposedPorts) != 0 || len(sig.AptPackages) != 0 || len(sig.Stages) != 0 || sig.EntryCmd != "" || len(sig.InferredDeps) != 0 {
			t.Errorf("expected empty signals, got %+v", sig)
		}
	})

	t.Run("empty_dockerfile", func(t *testing.T) {
		sig := ParseDockerfile([]byte(""))
		if len(sig.ExposedPorts) != 0 {
			t.Errorf("expected no ports, got %v", sig.ExposedPorts)
		}
	})

	t.Run("apt_get_with_chained_commands", func(t *testing.T) {
		sig := ParseDockerfile([]byte("FROM ubuntu\nRUN apt-get update && apt-get install -y postgresql-client && rm -rf /var/lib/apt/lists/*\n"))
		if len(sig.AptPackages) != 1 || sig.AptPackages[0] != "postgresql-client" {
			t.Errorf("expected [postgresql-client], got %v", sig.AptPackages)
		}
	})
}
