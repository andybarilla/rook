package cli_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/andybarilla/flock/internal/cli"
	"github.com/andybarilla/flock/internal/databases"
)

func TestRenderServiceStatus(t *testing.T) {
	services := []databases.ServiceInfo{
		{Type: "mysql", Enabled: true, Running: true, Port: 3306},
		{Type: "postgresql", Enabled: true, Running: false, Port: 5432},
		{Type: "redis", Enabled: false, Running: false, Port: 6379},
	}

	var buf bytes.Buffer
	cli.RenderServiceStatus(&buf, services, false)
	out := buf.String()

	if !bytes.Contains([]byte(out), []byte("mysql")) {
		t.Errorf("output missing mysql: %s", out)
	}
	if !bytes.Contains([]byte(out), []byte("SERVICE")) {
		t.Errorf("output missing SERVICE header: %s", out)
	}
}

func TestRenderServiceStatusJSON(t *testing.T) {
	services := []databases.ServiceInfo{
		{Type: "mysql", Enabled: true, Running: true, Port: 3306},
	}

	var buf bytes.Buffer
	cli.RenderServiceStatus(&buf, services, true)

	var result []databases.ServiceInfo
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 service, got %d", len(result))
	}
	if string(result[0].Type) != "mysql" {
		t.Errorf("type = %q, want mysql", result[0].Type)
	}
}

func TestRenderServiceStatusEmpty(t *testing.T) {
	var buf bytes.Buffer
	cli.RenderServiceStatus(&buf, nil, false)
	out := buf.String()

	if !bytes.Contains([]byte(out), []byte("No services")) {
		t.Errorf("expected 'No services' message, got: %s", out)
	}
}
