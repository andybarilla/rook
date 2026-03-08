package cli_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/andybarilla/flock/internal/cli"
	"github.com/andybarilla/flock/internal/registry"
)

func TestListCmdTable(t *testing.T) {
	sites := []registry.Site{
		{Path: "/home/user/myapp", Domain: "myapp.test", PHPVersion: "8.3", TLS: true},
		{Path: "/home/user/api", Domain: "api.test", NodeVersion: "20", TLS: false},
	}

	var buf bytes.Buffer
	cli.RenderSiteList(&buf, sites, false)
	out := buf.String()

	if !bytes.Contains([]byte(out), []byte("myapp.test")) {
		t.Errorf("output missing myapp.test: %s", out)
	}
	if !bytes.Contains([]byte(out), []byte("api.test")) {
		t.Errorf("output missing api.test: %s", out)
	}
	if !bytes.Contains([]byte(out), []byte("DOMAIN")) {
		t.Errorf("output missing DOMAIN header: %s", out)
	}
}

func TestListCmdJSON(t *testing.T) {
	sites := []registry.Site{
		{Path: "/home/user/myapp", Domain: "myapp.test", PHPVersion: "8.3", TLS: true},
	}

	var buf bytes.Buffer
	cli.RenderSiteList(&buf, sites, true)

	var result []registry.Site
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 site, got %d", len(result))
	}
	if result[0].Domain != "myapp.test" {
		t.Errorf("domain = %q, want myapp.test", result[0].Domain)
	}
}

func TestListCmdEmptyTable(t *testing.T) {
	var buf bytes.Buffer
	cli.RenderSiteList(&buf, nil, false)
	out := buf.String()

	if !bytes.Contains([]byte(out), []byte("No sites")) {
		t.Errorf("expected 'No sites' message, got: %s", out)
	}
}

func TestListCmdEmptyJSON(t *testing.T) {
	var buf bytes.Buffer
	cli.RenderSiteList(&buf, nil, true)

	var result []registry.Site
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected 0 sites, got %d", len(result))
	}
}

func TestRenderAddJSON(t *testing.T) {
	var buf bytes.Buffer
	site := registry.Site{Path: "/home/user/myapp", Domain: "myapp.test"}
	cli.FormatJSON(&buf, site)

	var result registry.Site
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result.Domain != "myapp.test" {
		t.Errorf("domain = %q, want myapp.test", result.Domain)
	}
}

func TestRenderRemoveJSON(t *testing.T) {
	var buf bytes.Buffer
	cli.FormatJSON(&buf, map[string]string{"removed": "myapp.test"})

	var result map[string]string
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["removed"] != "myapp.test" {
		t.Errorf("removed = %q, want myapp.test", result["removed"])
	}
}
