package discovery

import (
	"regexp"
	"strings"
)

// ScriptChange describes a sanitization change made to a script.
type ScriptChange struct {
	Description string
}

var whileConditionRe = regexp.MustCompile(`^\s*while\s+(.+);\s*do\s*$`)

// SanitizeScript removes devcontainer-specific patterns from shell scripts.
// Returns the sanitized content and a list of changes made.
func SanitizeScript(content []byte) ([]byte, []ScriptChange) {
	lines := strings.Split(string(content), "\n")
	var changes []ScriptChange

	// Rule 1: Remove wait loops (while/sleep/done blocks with preceding comments/echos)
	lines, changes = removeWaitLoops(lines, changes)

	// Rule 2: Remove keep-alive commands
	var removedKeepAlive bool
	lines, changes, removedKeepAlive = removeKeepAlive(lines, changes)

	// Rule 3: Strip background operators when keep-alive was removed
	if removedKeepAlive {
		lines, changes = stripBackground(lines, changes)
	}

	// Rule 4: Collapse consecutive blank lines (only when changes were made)
	if len(changes) > 0 {
		lines = collapseBlankLines(lines)
	}

	return []byte(strings.Join(lines, "\n")), changes
}

func collapseBlankLines(lines []string) []string {
	var result []string
	prevBlank := false
	for _, line := range lines {
		blank := strings.TrimSpace(line) == ""
		if blank && prevBlank {
			continue
		}
		prevBlank = blank
		result = append(result, line)
	}
	// Remove trailing blank lines
	for len(result) > 0 && strings.TrimSpace(result[len(result)-1]) == "" {
		result = result[:len(result)-1]
	}
	// Ensure trailing newline
	if len(result) > 0 {
		result = append(result, "")
	}
	return result
}

var keepAlivePatterns = []string{
	"exec sleep infinity",
	"sleep infinity",
	"exec tail -f /dev/null",
	"tail -f /dev/null",
}

func removeKeepAlive(lines []string, changes []ScriptChange) ([]string, []ScriptChange, bool) {
	var result []string
	removed := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		isKeepAlive := false
		for _, pat := range keepAlivePatterns {
			if trimmed == pat {
				isKeepAlive = true
				break
			}
		}
		if isKeepAlive {
			// Remove preceding contiguous comment and blank lines
			for len(result) > 0 {
				t := strings.TrimSpace(result[len(result)-1])
				if strings.HasPrefix(t, "#") || t == "" {
					result = result[:len(result)-1]
				} else {
					break
				}
			}
			changes = append(changes, ScriptChange{
				Description: "Removed keep-alive command (" + trimmed + ")",
			})
			removed = true
			continue
		}
		result = append(result, line)
	}
	return result, changes, removed
}

func stripBackground(lines []string, changes []ScriptChange) ([]string, []ScriptChange) {
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Strip trailing & from commands
		if strings.HasSuffix(trimmed, " &") && !strings.HasPrefix(trimmed, "#") {
			cmd := strings.TrimSuffix(trimmed, " &")
			indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
			result = append(result, indent+cmd)
			changes = append(changes, ScriptChange{
				Description: "Removed background operator from '" + cmd + "'",
			})
			continue
		}
		// Clean "in the background" from comments
		if strings.HasPrefix(trimmed, "#") && strings.Contains(trimmed, "in the background") {
			cleaned := strings.Replace(line, " in the background", "", 1)
			result = append(result, cleaned)
			continue
		}
		result = append(result, line)
	}
	return result, changes
}

// removeWaitLoops removes while loops whose body is only sleep commands,
// along with contiguous preceding comment and echo lines.
func removeWaitLoops(lines []string, changes []ScriptChange) ([]string, []ScriptChange) {
	var result []string
	i := 0
	for i < len(lines) {
		m := whileConditionRe.FindStringSubmatch(strings.TrimRight(lines[i], "\r"))
		if m != nil {
			// Check if body is only sleep lines, ending with done
			bodyStart := i + 1
			bodyEnd := -1
			onlySleep := true
			for j := bodyStart; j < len(lines); j++ {
				trimmed := strings.TrimSpace(lines[j])
				if trimmed == "done" {
					bodyEnd = j
					break
				}
				if !strings.HasPrefix(trimmed, "sleep ") && trimmed != "" {
					onlySleep = false
					break
				}
			}
			if bodyEnd != -1 && onlySleep {
				// Remove preceding contiguous comment and wait-echo lines
				for len(result) > 0 {
					trimmed := strings.TrimSpace(result[len(result)-1])
					isComment := strings.HasPrefix(trimmed, "#")
					isWaitEcho := strings.HasPrefix(trimmed, "echo ") && strings.Contains(strings.ToLower(trimmed), "wait")
					if isComment || isWaitEcho {
						result = result[:len(result)-1]
					} else {
						break
					}
				}
				condition := strings.TrimSpace(m[1])
				changes = append(changes, ScriptChange{
					Description: "Removed wait loop (while " + condition + ")",
				})
				i = bodyEnd + 1
				continue
			}
		}
		result = append(result, lines[i])
		i++
	}
	return result, changes
}
