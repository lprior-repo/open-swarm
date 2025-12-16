package dag

import (
	"fmt"
	"github.com/gammazero/toposort"
)

// Scheduler handles dependency resolution
type Scheduler struct{}

// BuildExecutionOrder performs topological sort on tasks.
// Returns a flat list of task names in safe execution order.
func (s *Scheduler) BuildExecutionOrder(tasks []Task) ([]string, error) {
	if len(tasks) == 0 {
		return []string{}, nil
	}

	// Build edges from dependencies
	edges := make([]toposort.Edge, 0)
	for _, t := range tasks {
		for _, dep := range t.Deps {
			edges = append(edges, toposort.Edge{dep, t.Name})
		}
	}

	// Optimization: If no edges, return simple order
	if len(edges) == 0 {
		flatOrder := make([]string, 0, len(tasks))
		for _, t := range tasks {
			flatOrder = append(flatOrder, t.Name)
		}
		return flatOrder, nil
	}

	sortedNodes, err := toposort.Toposort(edges)
	if err != nil {
		return nil, fmt.Errorf("cycle detected in DAG: %w", err)
	}

	// Reconstruct full list ensuring disconnected roots are included
	inSorted := make(map[string]bool, len(sortedNodes))
	flatOrder := make([]string, 0, len(tasks))

	for _, node := range sortedNodes {
		name := node.(string)
		inSorted[name] = true
		flatOrder = append(flatOrder, name)
	}

	// Prepend tasks that were not part of the dependency graph (roots)
	for _, t := range tasks {
		if !inSorted[t.Name] {
			flatOrder = append([]string{t.Name}, flatOrder...)
		}
	}

	return flatOrder, nil
}
