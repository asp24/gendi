package ir

import (
	"slices"

	di "github.com/asp24/gendi"
)

type sharedOptimizerPhase struct{}

// Apply identifies shared services that sit on a single-parent chain and marks them as
// non-shared when the chain is anchored by a shared ancestor. Chains that terminate at a
// public non-shared service stop at that boundary (the direct child stays shared). The
// optimization is applied recursively from the bottom of the dependency tree (leaves) upwards.
func (p *sharedOptimizerPhase) Apply(_ *di.Config, container *Container) error {
	// Map service ID to list of parent service IDs (referencing services)
	usage := make(map[string][]string)

	for _, service := range container.Services {
		// Dependencies contains resolved direct dependencies.
		// We use it to Apply the reverse graph (Usage).
		for _, dep := range service.Dependencies {
			// Track unique parents. Dependencies should be unique per service,
			// but we check existence to be safe.
			if !slices.Contains(usage[dep.ID], service.ID) {
				usage[dep.ID] = append(usage[dep.ID], service.ID)
			}
		}
	}

	// Process in post-order: deshare children (C) before parents (B) in a chain A->B->C
	for svc := range container.ServicesPostOrder() {
		if p.canDeshare(container, svc, usage) {
			svc.Shared = false
		}
	}

	return nil
}

func (p *sharedOptimizerPhase) canDeshare(container *Container, svc *Service, usage map[string][]string) bool {
	// Candidate must be Shared and not Public and not Alias
	if !svc.Shared || svc.Public || svc.IsAlias() {
		return false
	}

	curr := svc
	for {
		parents := usage[curr.ID]
		if len(parents) != 1 {
			return false
		}

		parentID := parents[0]
		parent := container.Services[parentID]
		if parent.Shared {
			return true
		}

		// Stop if parent is public but non-shared: its direct child keeps shared semantics.
		if parent.Public {
			return false
		}

		// Parent is Non-Shared, move up the chain.
		curr = parent
	}
}
