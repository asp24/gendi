package ir

import "sort"

// pruneUnreachable removes services not reachable from public services or tags.
func pruneUnreachable(ctx *buildContext) {
	reachable := map[string]bool{}
	var queue []*Service

	for _, svc := range ctx.services {
		if svc != nil && svc.Public {
			if !reachable[svc.ID] {
				reachable[svc.ID] = true
				queue = append(queue, svc)
			}
		}
	}
	for _, tag := range ctx.tags {
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

	for id := range ctx.services {
		if !reachable[id] {
			delete(ctx.services, id)
		}
	}

	ctx.order = ctx.order[:0]
	for id := range ctx.services {
		ctx.order = append(ctx.order, id)
	}
	sort.Strings(ctx.order)

	for _, tag := range ctx.tags {
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
