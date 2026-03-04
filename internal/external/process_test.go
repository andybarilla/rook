package external

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestExecProcessStarter(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}

	// Create a tiny script that reads one line from stdin and echoes a JSON-RPC response
	dir := t.TempDir()
	script := `#!/bin/sh
read line
echo '{"jsonrpc":"2.0","id":1,"result":{}}'
`
	scriptPath := filepath.Join(dir, "test-plugin")
	os.WriteFile(scriptPath, []byte(script), 0o755)

	proc, err := ExecProcessStarter(scriptPath)
	if err != nil {
		t.Fatalf("ExecProcessStarter failed: %v", err)
	}

	rpc := newRPCClient(proc.Stdout(), proc.Stdin())
	err = rpc.Call("plugin.init", nil, nil)
	if err != nil {
		t.Fatalf("RPC call failed: %v", err)
	}

	proc.Kill()
	proc.Wait()
}
