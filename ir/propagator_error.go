package ir

import (
	di "github.com/asp24/gendi"
)

// errorPropagator propagates error flags through the dependency graph
type errorPropagator struct{}

// propagate computes error propagation for all services in O(n) time
// using topological ordering to avoid iterative convergence.
func (p *errorPropagator) propagate(_ *di.Config, container *Container) {
	// Process services in topological order (dependencies before dependents)
	for svc := range container.ServicesPostOrder() {
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

		// CanError mirrors BuildCanError after dependency propagation
		svc.CanError = svc.BuildCanError
	}
}
