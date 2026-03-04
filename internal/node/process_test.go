package node_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/andybarilla/flock/internal/node"
)

func TestProcessRunnerStartAndStop(t *testing.T) {
	// Create a temp dir with a package.json that runs a simple HTTP server
	dir := t.TempDir()
	packageJSON := `{"name":"test","scripts":{"start":"node server.js"}}`
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(packageJSON), 0o644)

	// Simple Node server that listens on PORT env var
	serverJS := `
const http = require('http');
const port = process.env.PORT || 3000;
const server = http.createServer((req, res) => {
	res.writeHead(200);
	res.end('ok');
});
server.listen(port, '127.0.0.1');
`
	os.WriteFile(filepath.Join(dir, "server.js"), []byte(serverJS), 0o644)

	runner := node.NewProcessRunner()
	port := 13100 // use high port to avoid conflicts

	err := runner.StartApp(dir, port)
	if err != nil {
		t.Fatalf("StartApp: %v", err)
	}

	// Give the process a moment to start
	time.Sleep(500 * time.Millisecond)

	if !runner.IsRunning(dir) {
		t.Error("expected app to be running")
	}
	if runner.AppPort(dir) != port {
		t.Errorf("AppPort = %d, want %d", runner.AppPort(dir), port)
	}

	err = runner.StopApp(dir)
	if err != nil {
		t.Fatalf("StopApp: %v", err)
	}

	if runner.IsRunning(dir) {
		t.Error("expected app to be stopped")
	}
}

func TestProcessRunnerStopNonexistent(t *testing.T) {
	runner := node.NewProcessRunner()
	err := runner.StopApp("/no/such/dir")
	if err == nil {
		t.Error("expected error stopping nonexistent app")
	}
}

func TestProcessRunnerIsRunningFalseByDefault(t *testing.T) {
	runner := node.NewProcessRunner()
	if runner.IsRunning("/no/such/dir") {
		t.Error("expected IsRunning to be false for unknown dir")
	}
}

func TestProcessRunnerAppPortZeroByDefault(t *testing.T) {
	runner := node.NewProcessRunner()
	if runner.AppPort("/no/such/dir") != 0 {
		t.Error("expected AppPort to be 0 for unknown dir")
	}
}
