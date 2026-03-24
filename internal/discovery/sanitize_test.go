package discovery

import (
	"strings"
	"testing"
)

func TestSanitizeScript(t *testing.T) {
	t.Run("removes_keep_alive_with_comments", func(t *testing.T) {
		input := `#!/bin/bash
echo "starting"

# Keep the container alive
exec sleep infinity
`
		out, changes := SanitizeScript([]byte(input))
		result := string(out)

		if strings.Contains(result, "sleep infinity") {
			t.Error("expected sleep infinity to be removed")
		}
		if strings.Contains(result, "Keep the container") {
			t.Error("expected preceding comment to be removed")
		}
		if !strings.Contains(result, "echo \"starting\"") {
			t.Error("expected unrelated lines to be preserved")
		}
		if len(changes) != 1 {
			t.Fatalf("expected 1 change, got %d", len(changes))
		}
		if !strings.Contains(changes[0].Description, "keep-alive") {
			t.Errorf("expected change to mention keep-alive, got %q", changes[0].Description)
		}
	})

	t.Run("removes_tail_keepalive", func(t *testing.T) {
		input := `#!/bin/bash
do-stuff
tail -f /dev/null
`
		out, changes := SanitizeScript([]byte(input))
		result := string(out)

		if strings.Contains(result, "tail") {
			t.Error("expected tail -f /dev/null to be removed")
		}
		if len(changes) != 1 {
			t.Fatalf("expected 1 change, got %d", len(changes))
		}
	})

	t.Run("removes_wait_loop_with_comments", func(t *testing.T) {
		input := `#!/bin/bash
cd /workspaces/app

# Wait for post-create.sh to finish (it creates this marker file)
echo "Waiting for post-create to finish..."
while [ ! -f /tmp/.devcontainer-ready ]; do
  sleep 1
done

echo "ready"
`
		out, changes := SanitizeScript([]byte(input))
		result := string(out)

		if strings.Contains(result, "while") {
			t.Error("expected while loop to be removed")
		}
		if strings.Contains(result, "devcontainer-ready") {
			t.Error("expected devcontainer-ready references to be removed")
		}
		if !strings.Contains(result, "echo \"ready\"") {
			t.Error("expected unrelated echo to be preserved")
		}
		if len(changes) != 1 {
			t.Fatalf("expected 1 change, got %d", len(changes))
		}
		if !strings.Contains(changes[0].Description, "wait loop") {
			t.Errorf("expected change description to mention wait loop, got %q", changes[0].Description)
		}
	})
}
