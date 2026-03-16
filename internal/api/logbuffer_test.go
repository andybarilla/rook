package api_test

import (
	"testing"

	"github.com/andybarilla/rook/internal/api"
)

func TestAddAndGet(t *testing.T) {
	buf := api.NewLogBuffer(100)
	buf.Add("ws1", "svc1", "hello")
	buf.Add("ws1", "svc1", "world")

	lines := buf.Get("ws1", "", 0)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lines[0].Line != "hello" || lines[1].Line != "world" {
		t.Fatalf("unexpected lines: %v", lines)
	}
}

func TestFilterByService(t *testing.T) {
	buf := api.NewLogBuffer(100)
	buf.Add("ws1", "svc1", "line1")
	buf.Add("ws1", "svc2", "line2")
	buf.Add("ws1", "svc1", "line3")

	lines := buf.Get("ws1", "svc1", 0)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines for svc1, got %d", len(lines))
	}
	if lines[0].Line != "line1" || lines[1].Line != "line3" {
		t.Fatalf("unexpected lines: %v", lines)
	}
}

func TestRollingLimit(t *testing.T) {
	buf := api.NewLogBuffer(3)
	for i := 0; i < 5; i++ {
		buf.Add("ws1", "svc1", "line")
	}

	lines := buf.Get("ws1", "", 0)
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines (rolling), got %d", len(lines))
	}
}

func TestLinesLimit(t *testing.T) {
	buf := api.NewLogBuffer(100)
	for i := 0; i < 10; i++ {
		buf.Add("ws1", "svc1", "line")
	}

	lines := buf.Get("ws1", "", 5)
	if len(lines) != 5 {
		t.Fatalf("expected 5 lines (limit), got %d", len(lines))
	}
}

func TestHasTimestamp(t *testing.T) {
	buf := api.NewLogBuffer(100)
	entry := buf.Add("ws1", "svc1", "test")
	if entry.Timestamp.IsZero() {
		t.Fatal("expected non-zero timestamp")
	}
}

func TestMultipleWorkspaces(t *testing.T) {
	buf := api.NewLogBuffer(100)
	buf.Add("ws1", "svc1", "line1")
	buf.Add("ws2", "svc1", "line2")

	lines1 := buf.Get("ws1", "", 0)
	lines2 := buf.Get("ws2", "", 0)

	if len(lines1) != 1 || len(lines2) != 1 {
		t.Fatalf("expected 1 line each, got %d and %d", len(lines1), len(lines2))
	}
	if lines1[0].Line != "line1" || lines2[0].Line != "line2" {
		t.Fatal("unexpected lines for multiple workspaces")
	}
}
