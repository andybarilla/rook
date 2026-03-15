package discovery

import "github.com/andybarilla/rook/internal/workspace"

// DiscoveryResult holds the results of running one or more discoverers.
type DiscoveryResult struct {
	Source   string
	Services map[string]workspace.Service
	Groups   map[string][]string
}

// Discoverer detects and extracts service information from project configuration files.
type Discoverer interface {
	Name() string
	Detect(dir string) bool
	Discover(dir string) (*DiscoveryResult, error)
}

// RunAll runs all provided discoverers against the given directory,
// merging their results into a single DiscoveryResult.
func RunAll(dir string, discoverers []Discoverer) (*DiscoveryResult, error) {
	merged := &DiscoveryResult{
		Services: make(map[string]workspace.Service),
		Groups:   make(map[string][]string),
	}
	for _, d := range discoverers {
		if !d.Detect(dir) {
			continue
		}
		result, err := d.Discover(dir)
		if err != nil {
			return nil, err
		}
		for name, svc := range result.Services {
			merged.Services[name] = svc
		}
		for name, group := range result.Groups {
			merged.Groups[name] = group
		}
		if merged.Source == "" {
			merged.Source = d.Name()
		} else {
			merged.Source += ", " + d.Name()
		}
	}
	return merged, nil
}
