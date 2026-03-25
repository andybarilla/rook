package runner

import (
	"io"
	"os/exec"
	"strings"
	"testing"
)

func TestCmdReadCloser_Read(t *testing.T) {
	inner := io.NopCloser(strings.NewReader("hello\nworld\n"))
	crc := &cmdReadCloser{ReadCloser: inner}
	data, err := io.ReadAll(crc)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello\nworld\n" {
		t.Fatalf("got %q", string(data))
	}
}

func TestCmdReadCloser_CloseWaitsCmd(t *testing.T) {
	cmd := exec.Command("echo", "done")
	cmd.Start()
	inner := io.NopCloser(strings.NewReader(""))
	crc := &cmdReadCloser{ReadCloser: inner, cmd: cmd}
	if err := crc.Close(); err != nil {
		t.Fatal(err)
	}
	// cmd.Wait already called by Close — calling again should error
	if err := cmd.Wait(); err == nil {
		t.Fatal("expected error from double Wait")
	}
}

func TestCmdReadCloser_CloseNilCmd(t *testing.T) {
	inner := io.NopCloser(strings.NewReader(""))
	crc := &cmdReadCloser{ReadCloser: inner}
	if err := crc.Close(); err != nil {
		t.Fatal(err)
	}
}
