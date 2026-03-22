package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/andybarilla/rook/internal/workspace"
)

// formatRebuildPrompt writes the grouped stale-build prompt to w.
// resolvedServices is the list of service names in the active profile;
// if nil, all services in the workspace are considered.
func formatRebuildPrompt(w io.Writer, staleServices map[string][]string, services map[string]workspace.Service, resolvedServices []string) {
	// Build set of resolved service names for filtering
	resolvedSet := make(map[string]bool)
	if resolvedServices != nil {
		for _, name := range resolvedServices {
			resolvedSet[name] = true
		}
	}

	// Build reverse map: source -> sorted consumers (filtered by profile)
	consumers := make(map[string][]string)
	for name, svc := range services {
		if svc.BuildFrom == "" {
			continue
		}
		if _, stale := staleServices[svc.BuildFrom]; !stale {
			continue
		}
		if resolvedServices != nil && !resolvedSet[name] {
			continue
		}
		consumers[svc.BuildFrom] = append(consumers[svc.BuildFrom], name)
	}
	for source := range consumers {
		sort.Strings(consumers[source])
	}

	// Sort stale service names for deterministic output
	names := make([]string, 0, len(staleServices))
	for name := range staleServices {
		names = append(names, name)
	}
	sort.Strings(names)

	fmt.Fprintf(w, "%d service(s) need rebuild:\n", len(staleServices))
	for _, name := range names {
		reasons := staleServices[name]
		if len(reasons) > 0 {
			fmt.Fprintf(w, "  - %s (%s)\n", name, reasons[0])
		} else {
			fmt.Fprintf(w, "  - %s\n", name)
		}
		if deps := consumers[name]; len(deps) > 0 {
			fmt.Fprintf(w, "    also used by: %s\n", strings.Join(deps, ", "))
		}
	}
}

// buildFromConsumers returns sorted names of build_from consumers whose
// source has ForceBuild=true, filtered to only services in resolvedServices.
func buildFromConsumers(services map[string]workspace.Service, resolvedServices []string) []string {
	resolvedSet := make(map[string]bool)
	for _, name := range resolvedServices {
		resolvedSet[name] = true
	}

	var result []string
	for name, svc := range services {
		if svc.BuildFrom == "" {
			continue
		}
		if !resolvedSet[name] {
			continue
		}
		source, ok := services[svc.BuildFrom]
		if !ok || !source.ForceBuild {
			continue
		}
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}
