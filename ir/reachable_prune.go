package ir

import (
	di "github.com/asp24/gendi"
)

// pruneUnreachable removes services not reachable from public services.
// Note: After tag desugaring, public tags become public services with !tagged: prefix,
// so we only need to check Services, not tags.
func pruneUnreachable(_ *di.Config, container *Container) {
	reachable := map[string]bool{}
	var queue []*Service

	// Start from all public services (including desugared public tags)
	for _, svc := range container.Services {
		if svc != nil && svc.Public {
			if !reachable[svc.ID] {
				reachable[svc.ID] = true
				queue = append(queue, svc)
			}
		}
	}

	// BFS to find all reachable services
	for len(queue) > 0 {
		svc := queue[0]
		queue = queue[1:]
		if svc == nil {
			continue
		}
		for _, dep := range svc.Dependencies {
			if dep == nil || reachable[dep.ID] {
				continue
			}
			reachable[dep.ID] = true
			queue = append(queue, dep)
		}
	}

	// Remove unreachable services
	for id := range container.Services {
		if !reachable[id] {
			delete(container.Services, id)
		}
	}
}
