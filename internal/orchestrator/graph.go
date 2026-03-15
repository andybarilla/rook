package orchestrator

import (
	"fmt"

	"github.com/andybarilla/rook/internal/workspace"
)

// TopoSort performs a topological sort on the given services, returning them
// in dependency order (dependencies first). It detects circular dependencies
// and unknown service references.
func TopoSort(services map[string]workspace.Service, targets []string) ([]string, error) {
	const (
		unvisited = 0
		visiting  = 1
		visited   = 2
	)
	state := make(map[string]int)
	var order []string

	var visit func(name string) error
	visit = func(name string) error {
		switch state[name] {
		case visited:
			return nil
		case visiting:
			return fmt.Errorf("circular dependency detected involving %q", name)
		}
		state[name] = visiting
		svc, ok := services[name]
		if !ok {
			return fmt.Errorf("unknown service: %q", name)
		}
		for _, dep := range svc.DependsOn {
			if err := visit(dep); err != nil {
				return err
			}
		}
		state[name] = visited
		order = append(order, name)
		return nil
	}

	for _, name := range targets {
		if err := visit(name); err != nil {
			return nil, err
		}
	}
	return order, nil
}
