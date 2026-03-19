package envgen

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ParseEnvFile reads a .env file and returns key-value pairs.
// Supports: KEY=VALUE, comments (#), blank lines, quoted values,
// export prefix. Lines without = are skipped.
func ParseEnvFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("reading env file: %w", err)
	}
	defer f.Close()

	result := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")

		idx := strings.Index(line, "=")
		if idx < 0 {
			continue
		}

		key := line[:idx]
		value := line[idx+1:]

		// Strip matching surrounding quotes
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		result[key] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading env file: %w", err)
	}
	return result, nil
}

// LoadProcessEnvFile reads an env file, expands shell vars, resolves
// templates (using localhost + allocated ports for process services),
// and merges with inline environment. Inline values take precedence.
func LoadProcessEnvFile(path string, inlineEnv map[string]string, portMap map[string]int) (map[string]string, error) {
	fileVars, err := ParseEnvFile(path)
	if err != nil {
		return nil, err
	}

	// Expand shell vars in each value
	for k, v := range fileVars {
		fileVars[k] = ExpandShellVars(v)
	}

	// Resolve templates (process services use localhost)
	fileVars, err = ResolveTemplates(fileVars, portMap)
	if err != nil {
		return nil, fmt.Errorf("resolving env file templates: %w", err)
	}

	// Merge: start with file vars, overlay inline (inline wins)
	result := fileVars
	for k, v := range inlineEnv {
		result[k] = v
	}

	return result, nil
}
