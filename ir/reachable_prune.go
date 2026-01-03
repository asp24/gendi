package ir

import (
	di "github.com/asp24/gendi"
)

// pruneUnreachable removes services not reachable from public services or tags.
func pruneUnreachable(_ *di.Config, container *Container) {
	reachable := map[string]bool{}
	var queue []*Service

	for _, svc := range container.Services {
		if svc != nil && svc.Public {
			if !reachable[svc.ID] {
				reachable[svc.ID] = true
				queue = append(queue, svc)
			}
		}
	}
	for _, tag := range container.Tags {
		if tag == nil || !tag.Public {
			continue
		}
		for _, svc := range tag.Services {
			if svc == nil || reachable[svc.ID] {
				continue
			}
			reachable[svc.ID] = true
			queue = append(queue, svc)
		}
	}

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

	for id := range container.Services {
		if !reachable[id] {
			delete(container.Services, id)
		}
	}

	for _, tag := range container.Tags {
		if tag == nil {
			continue
		}
		out := tag.Services[:0]
		for _, svc := range tag.Services {
			if svc != nil && reachable[svc.ID] {
				out = append(out, svc)
			}
		}
		tag.Services = out
	}
}
