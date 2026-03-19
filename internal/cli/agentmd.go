package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/andybarilla/rook/internal/workspace"
)

// ensureAgentMDRookSection appends a rook section to an existing CLAUDE.md or
// AGENTS.md file. It prefers CLAUDE.md if both exist. If neither exists or the
// file already contains a <!-- rook --> tag, it does nothing.
func ensureAgentMDRookSection(dir string, m *workspace.Manifest) {
	// Try CLAUDE.md first, then AGENTS.md
	var target string
	for _, name := range []string{"CLAUDE.md", "AGENTS.md"} {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			target = p
			break
		}
	}
	if target == "" {
		return
	}

	content, err := os.ReadFile(target)
	if err != nil {
		return
	}

	if strings.Contains(string(content), "<!-- rook -->") {
		return
	}

	section := buildRookSection(m)

	// Ensure we start on a new line
	s := string(content)
	if len(s) > 0 && !strings.HasSuffix(s, "\n") {
		s += "\n"
	}
	s += "\n" + section

	os.WriteFile(target, []byte(s), 0644)
}

func buildRookSection(m *workspace.Manifest) string {
	var b strings.Builder
	b.WriteString("<!-- rook -->\n")
	b.WriteString("## Rook Workspace\n\n")
	fmt.Fprintf(&b, "This project is managed by [Rook](https://github.com/andybarilla/rook), workspace name: `%s`.\n\n", m.Name)

	// List services sorted for deterministic output
	names := make([]string, 0, len(m.Services))
	for name := range m.Services {
		names = append(names, name)
	}
	sort.Strings(names)

	b.WriteString("### Services\n\n")
	for _, name := range names {
		svc := m.Services[name]
		if svc.Image != "" {
			fmt.Fprintf(&b, "- `%s` — %s\n", name, svc.Image)
		} else {
			fmt.Fprintf(&b, "- `%s` — process\n", name)
		}
	}

	b.WriteString("\n### Commands\n\n")
	b.WriteString("```bash\n")
	fmt.Fprintf(&b, "rook up %s              # Start all services\n", m.Name)
	fmt.Fprintf(&b, "rook down %s            # Stop all services\n", m.Name)
	fmt.Fprintf(&b, "rook status %s          # Show service status\n", m.Name)
	fmt.Fprintf(&b, "rook logs %s <service>  # Tail service logs\n", m.Name)
	b.WriteString("```\n")
	b.WriteString("<!-- /rook -->\n")

	return b.String()
}
