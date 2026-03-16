package cli

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestLogMux_FormatsLines(t *testing.T) {
	var out bytes.Buffer
	mux := newLogMux(&out)
	r := io.NopCloser(strings.NewReader("line 1\nline 2\n"))
	done := make(chan struct{})
	go func() { mux.addStream("postgres", r, 0); close(done) }()
	<-done
	output := out.String()
	if !strings.Contains(output, "[postgres") {
		t.Errorf("expected prefix, got:\n%s", output)
	}
	if !strings.Contains(output, "line 1") {
		t.Error("missing line 1")
	}
}

func TestLogMux_MultipleServices(t *testing.T) {
	var out bytes.Buffer
	mux := newLogMux(&out)
	r1 := io.NopCloser(strings.NewReader("pg ready\n"))
	r2 := io.NopCloser(strings.NewReader("app started\n"))
	done := make(chan struct{}, 2)
	go func() { mux.addStream("postgres", r1, 0); done <- struct{}{} }()
	go func() { mux.addStream("app", r2, 1); done <- struct{}{} }()
	<-done
	<-done
	output := out.String()
	if !strings.Contains(output, "pg ready") {
		t.Error("missing postgres")
	}
	if !strings.Contains(output, "app started") {
		t.Error("missing app")
	}
}
