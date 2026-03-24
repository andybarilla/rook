package envgen_test

import (
	"testing"

	"github.com/andybarilla/rook/internal/envgen"
)

func TestRewrite_URLWithHostAndPort(t *testing.T) {
	result, err := envgen.Rewrite("postgres://user:pass@localhost:5432/db", "postgres")
	if err != nil {
		t.Fatal(err)
	}
	expected := "postgres://user:pass@{{.Host.postgres}}:{{.Port.postgres}}/db"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestRewrite_URLWithHostOnly(t *testing.T) {
	result, err := envgen.Rewrite("http://localhost/api", "app")
	if err != nil {
		t.Fatal(err)
	}
	expected := "http://{{.Host.app}}/api"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestRewrite_URLWithIPAndPort(t *testing.T) {
	result, err := envgen.Rewrite("redis://127.0.0.1:6379/0", "redis")
	if err != nil {
		t.Fatal(err)
	}
	expected := "redis://{{.Host.redis}}:{{.Port.redis}}/0"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestRewrite_URLWithNonStandardPort(t *testing.T) {
	result, err := envgen.Rewrite("postgres://user:pass@localhost:9999/db", "postgres")
	if err != nil {
		t.Fatal(err)
	}
	expected := "postgres://user:pass@{{.Host.postgres}}:{{.Port.postgres}}/db"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestRewrite_URLWithQueryAndFragment(t *testing.T) {
	result, err := envgen.Rewrite("postgres://user:pass@localhost:5432/db?sslmode=disable#pool", "postgres")
	if err != nil {
		t.Fatal(err)
	}
	expected := "postgres://user:pass@{{.Host.postgres}}:{{.Port.postgres}}/db?sslmode=disable#pool"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestRewrite_URLWithEmptyHost(t *testing.T) {
	_, err := envgen.Rewrite("http:///path", "app")
	if err == nil {
		t.Error("expected error for URL with empty host and no port")
	}
}
