package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/andybarilla/rook/internal/workspace"
)

// ensureAgentMDRookSection upserts a rook section in an existing CLAUDE.md or
// AGENTS.md file. It prefers CLAUDE.md if both exist. If neither exists, it
// does nothing. Returns the action taken ("added", "updated", or "") and any error.
func ensureAgentMDRookSection(dir string, m *workspace.Manifest) (string, error) {
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
		return "", nil
	}

	content, err := os.ReadFile(target)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", filepath.Base(target), err)
	}

	s := string(content)
	section := buildRookSection(m)

	openTag := "<!-- rook -->"
	closeTag := "<!-- /rook -->\n"

	startIdx := strings.Index(s, openTag)
	if startIdx >= 0 {
		// Replace existing section
		endIdx := strings.Index(s, closeTag)
		if endIdx < 0 {
			return "", fmt.Errorf("found %s without matching <!-- /rook --> in %s", openTag, filepath.Base(target))
		}
		result := s[:startIdx] + section + s[endIdx+len(closeTag):]
		if err := os.WriteFile(target, []byte(result), 0644); err != nil {
			return "", fmt.Errorf("writing %s: %w", filepath.Base(target), err)
		}
		return "updated", nil
	}

	// Append new section
	if len(s) > 0 && !strings.HasSuffix(s, "\n") {
		s += "\n"
	}
	s += "\n" + section

	if err := os.WriteFile(target, []byte(s), 0644); err != nil {
		return "", fmt.Errorf("writing %s: %w", filepath.Base(target), err)
	}
	return "added", nil
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
		} else if svc.Build != "" {
			fmt.Fprintf(&b, "- `%s` — build (%s)\n", name, svc.Build)
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
