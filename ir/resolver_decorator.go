package ir

import (
	"fmt"
	"sort"
)

// decoratorResolver links decorators to their base services
type decoratorResolver struct{}

// resolve links all decorators to base services and sorts by priority
func (r *decoratorResolver) resolve(ctx *buildContext) error {
	for _, svc := range ctx.services {
		cfg := ctx.cfg.Services[svc.ID]
		if cfg.Decorates == "" {
			continue
		}

		base, ok := ctx.services[cfg.Decorates]
		if !ok {
			return fmt.Errorf("decorator %q decorates unknown service %q", svc.ID, cfg.Decorates)
		}

		svc.Decorates = base
		base.Decorators = append(base.Decorators, svc)
	}

	// Sort decorators by priority
	for _, svc := range ctx.services {
		if len(svc.Decorators) > 1 {
			sort.Slice(svc.Decorators, func(i, j int) bool {
				return svc.Decorators[i].Priority < svc.Decorators[j].Priority
			})
		}
	}

	return nil
}
