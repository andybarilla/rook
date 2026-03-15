package profile

import (
	"fmt"

	"github.com/andybarilla/rook/internal/workspace"
)

// Resolve expands a named profile into a deduplicated list of service names.
// It handles group expansion, wildcard ("*") entries, and an implicit "all"
// profile that returns every service when no explicit "all" profile is defined.
func Resolve(ws workspace.Workspace, profileName string) ([]string, error) {
	var entries []string

	if profileName == "all" {
		if p, ok := ws.Profiles["all"]; ok {
			entries = p
		} else {
			return ws.ServiceNames(), nil
		}
	} else {
		p, ok := ws.Profiles[profileName]
		if !ok {
			return nil, fmt.Errorf("unknown profile: %q", profileName)
		}
		entries = p
	}

	seen := make(map[string]bool)
	var result []string

	for _, entry := range entries {
		if entry == "*" {
			for _, name := range ws.ServiceNames() {
				if !seen[name] {
					seen[name] = true
					result = append(result, name)
				}
			}
			continue
		}

		if group, ok := ws.Groups[entry]; ok {
			for _, name := range group {
				if !seen[name] {
					seen[name] = true
					result = append(result, name)
				}
			}
			continue
		}

		if _, ok := ws.Services[entry]; ok {
			if !seen[entry] {
				seen[entry] = true
				result = append(result, entry)
			}
			continue
		}

		return nil, fmt.Errorf("profile %q references unknown service or group: %q", profileName, entry)
	}

	return result, nil
}
