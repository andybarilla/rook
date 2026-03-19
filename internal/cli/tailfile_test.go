package cli

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTailFile_ReadsExistingContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")
	os.WriteFile(path, []byte("existing line\n"), 0644)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reader, err := tailFile(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	done := make(chan string, 1)
	go func() {
		buf := make([]byte, 1024)
		n, _ := reader.Read(buf)
		done <- string(buf[:n])
	}()

	select {
	case content := <-done:
		if !strings.Contains(content, "existing line") {
			t.Errorf("expected existing content, got %q", content)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout reading existing content")
	}
}

func TestTailFile_FollowsNewWrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")
	os.WriteFile(path, []byte("initial\n"), 0644)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reader, err := tailFile(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	// Read initial content
	buf := make([]byte, 1024)
	reader.Read(buf)

	// Append new content
	f, _ := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	f.WriteString("appended line\n")
	f.Close()

	// Read new content with timeout
	done := make(chan string, 1)
	go func() {
		buf := make([]byte, 1024)
		n, _ := reader.Read(buf)
		done <- string(buf[:n])
	}()

	select {
	case content := <-done:
		if !strings.Contains(content, "appended line") {
			t.Errorf("expected appended content, got %q", content)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout reading appended content")
	}
}

func TestTailFile_ContextCancellation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")
	os.WriteFile(path, []byte("data\n"), 0644)

	ctx, cancel := context.WithCancel(context.Background())
	reader, err := tailFile(ctx, path)
	if err != nil {
		t.Fatal(err)
	}

	// Read existing content first
	buf := make([]byte, 1024)
	reader.Read(buf)

	// Cancel and verify reader returns EOF
	cancel()
	time.Sleep(300 * time.Millisecond)

	_, err = io.ReadAll(reader)
	if err != nil && err != io.EOF {
		t.Errorf("expected EOF or nil after cancel, got %v", err)
	}
}
