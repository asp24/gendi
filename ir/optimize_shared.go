package ir

import (
	"slices"

	di "github.com/asp24/gendi"
)

type sharedOptimizer struct{}

func (p *sharedOptimizer) hasPublicTagOrTagInjection(tags []*ServiceTag) bool {
	for _, t := range tags {
		if t.Tag.Public {
			return true
		}

		if len(t.Tag.Services) > 0 {
			return true
		}
	}

	return false
}

// resolve identifies shared services that are used by exactly one other shared service
// and marks them as non-shared.
//
// This is a safe optimization because if a service S is used only by service P,
// and P is a singleton (Shared), then:
// - If S is Shared: P holds the only reference to the singleton S.
// - If S is Non-Shared: P creates a new instance of S. Since P is created once, S is created once.
// Thus, the lifecycle of S is effectively the same (bound to P), but we save the overhead
// of registering S as a shared instance in the container.
//
// The optimization is applied recursively from the bottom of the dependency tree (leaves) upwards.
// This ensures that if A -> B -> C, and A, B, C are shared:
// 1. C is checked. Parent B is Shared. C becomes Non-Shared.
// 2. B is checked. Parent A is Shared. B becomes Non-Shared.
func (p *sharedOptimizer) resolve(_ *di.Config, container *Container) error {
	// Map service ID to list of parent service IDs (referencing services)
	usage := make(map[string][]string)

	for _, parent := range container.Services {
		if parent == nil {
			continue
		}
		// Dependencies contains resolved direct dependencies.
		// We use it to build the reverse graph (Usage).
		for _, dep := range parent.Dependencies {
			if dep == nil {
				continue
			}
			// Track unique parents. Dependencies should be unique per service,
			// but we check existence to be safe.
			if !slices.Contains(usage[dep.ID], parent.ID) {
				usage[dep.ID] = append(usage[dep.ID], parent.ID)
			}
		}
	}

	// Process in post-order: optimize children (C) before parents (B) in a chain A->B->C
	for svc := range container.ServicesPostOrder() {
		p.optimize(container, svc, usage)
	}

	return nil
}

func (p *sharedOptimizer) optimize(container *Container, svc *Service, usage map[string][]string) {
	// Candidate must be Shared and not Public and not Alias
	if !svc.Shared || svc.Public || svc.IsAlias() {
		return
	}

	// Candidate must not be referenced by tags.
	if p.hasPublicTagOrTagInjection(svc.Tags) {
		return
	}

	// Check usage
	parents := usage[svc.ID]
	if len(parents) != 1 {
		return
	}

	parentID := parents[0]
	parent := container.Services[parentID]

	// The single parent must be a Shared service.
	// If parent is Non-Shared (prototype), it is created multiple times.
	// If S is Shared, all instances of P share the same S.
	// If S becomes Non-Shared, each P gets a new S.
	// This changes semantics, so we only optimize if parent is Shared.
	if parent == nil || !parent.Shared {
		return
	}

	// Apply optimization
	svc.Shared = false
}
