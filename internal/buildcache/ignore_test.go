package buildcache_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/andybarilla/rook/internal/buildcache"
)

func TestParseDockerignore_MissingReturnsDefault(t *testing.T) {
	dir := t.TempDir()
	patterns, err := buildcache.ParseDockerignore(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Should include default exclusions
	if len(patterns) == 0 {
		t.Error("expected default patterns")
	}
}

func TestParseDockerignore_ReadsFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".dockerignore"), []byte("*.log\ntmp/\n"), 0644); err != nil {
		t.Fatal(err)
	}

	patterns, err := buildcache.ParseDockerignore(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Should include file patterns + defaults
	found := false
	for _, p := range patterns {
		if p == "*.log" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected *.log pattern")
	}
}

func TestParseGitignore_MissingReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	patterns, err := buildcache.ParseGitignore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(patterns) != 0 {
		t.Errorf("expected empty patterns for missing .gitignore, got %d", len(patterns))
	}
}

func TestParseGitignore_ReadsFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("node_modules/\ndist/\n# comment\n\n*.log\n"), 0644); err != nil {
		t.Fatal(err)
	}

	patterns, err := buildcache.ParseGitignore(dir)
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{"node_modules/", "dist/", "*.log"}
	if len(patterns) != len(expected) {
		t.Fatalf("expected %d patterns, got %d: %v", len(expected), len(patterns), patterns)
	}
	for i, p := range expected {
		if patterns[i] != p {
			t.Errorf("pattern[%d]: got %q, want %q", i, patterns[i], p)
		}
	}
}

func TestMatchesPatterns_Simple(t *testing.T) {
	patterns := []string{"*.log", "tmp/"}

	if buildcache.MatchesPatterns("test.log", patterns) != true {
		t.Error("test.log should match")
	}
	if buildcache.MatchesPatterns("src/test.log", patterns) != true {
		t.Error("src/test.log should match")
	}
	if buildcache.MatchesPatterns("tmp/file.txt", patterns) != true {
		t.Error("tmp/file.txt should match")
	}
	if buildcache.MatchesPatterns("main.go", patterns) != false {
		t.Error("main.go should not match")
	}
}

func TestMatchesPatterns_Negation(t *testing.T) {
	patterns := []string{"*.log", "!important.log"}

	if buildcache.MatchesPatterns("test.log", patterns) != true {
		t.Error("test.log should match")
	}
	if buildcache.MatchesPatterns("important.log", patterns) != false {
		t.Error("important.log should not match (negated)")
	}
}
