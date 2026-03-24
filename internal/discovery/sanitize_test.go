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

	t.Run("strips_background_when_keepalive_removed", func(t *testing.T) {
		input := `#!/bin/bash
# Start dev servers in the background
make dev-servers &

exec sleep infinity
`
		out, changes := SanitizeScript([]byte(input))
		result := string(out)

		if strings.Contains(result, " &") {
			t.Error("expected trailing & to be stripped")
		}
		if !strings.Contains(result, "make dev-servers") {
			t.Error("expected command to be preserved without &")
		}
		if strings.Contains(result, "in the background") {
			t.Error("expected 'in the background' removed from comment")
		}
		hasBackground := false
		for _, c := range changes {
			if strings.Contains(c.Description, "background") {
				hasBackground = true
			}
		}
		if !hasBackground {
			t.Error("expected a change about background operator removal")
		}
	})

	t.Run("preserves_background_when_no_keepalive", func(t *testing.T) {
		input := `#!/bin/bash
make dev-servers &
do-other-stuff
`
		out, changes := SanitizeScript([]byte(input))
		result := string(out)

		if !strings.Contains(result, "make dev-servers &") {
			t.Error("expected & to be preserved when no keep-alive present")
		}
		if len(changes) != 0 {
			t.Errorf("expected no changes, got %d", len(changes))
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
