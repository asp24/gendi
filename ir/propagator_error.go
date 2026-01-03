package ir

import (
	di "github.com/asp24/gendi"
)

// errorPropagator propagates error flags through the dependency graph
type errorPropagator struct{}

// propagate computes error propagation for all services in O(n) time
// using topological ordering to avoid iterative convergence.
func (p *errorPropagator) propagate(_ *di.Config, container *Container) {
	// Build topological order based on dependency graph
	order := p.topologicalSort(container.Services)

	// Phase 1: Propagate BuildCanError in dependency order
	// Process services in topological order (dependencies before dependents)
	for _, id := range order {
		svc := container.Services[id]

		// Initialize from constructor
		if svc.Constructor != nil && svc.Constructor.ReturnsError {
			svc.BuildCanError = true
		}

		// Propagate from dependencies
		// If any dependency's getter can error, then building this service can error
		for _, dep := range svc.Dependencies {
			if dep.CanError {
				svc.BuildCanError = true
				break
			}
		}
	}

	// Phase 2: CanError mirrors BuildCanError after dependency propagation.
	for _, id := range order {
		svc := container.Services[id]
		svc.CanError = svc.BuildCanError
	}
}

// topologicalSort returns service IDs in topological order using DFS.
// Dependencies come before dependents in the result.
func (p *errorPropagator) topologicalSort(services map[string]*Service) []string {
	result := make([]string, 0, len(services))
	visited := make(map[string]bool)

	var visit func(svc *Service)
	visit = func(svc *Service) {
		if visited[svc.ID] {
			return
		}
		visited[svc.ID] = true

		// Visit dependencies first.
		for _, dep := range svc.Dependencies {
			visit(dep)
		}

		// Add this service after its dependencies
		result = append(result, svc.ID)
	}

	// Visit all services
	for _, svc := range services {
		visit(svc)
	}

	return result
}
