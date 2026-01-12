package ir

import (
	"cmp"
	"slices"

	di "github.com/asp24/gendi"
	"github.com/asp24/gendi/xmaps"
)

// dependencyBuilderPhase builds service dependency graph.
// This phase must run after tagDesugarPhase, so all TaggedArg are already
// rewritten to ServiceRefArg.
type dependencyBuilderPhase struct{}

// collectDependencies recursively collects dependencies from an argument
func (r *dependencyBuilderPhase) collectDependencies(arg *Argument, deps map[string]*Service) {
	switch arg.Kind {
	case ServiceRefArg:
		if arg.Service != nil {
			deps[arg.Service.ID] = arg.Service
		}
	case SpreadArg:
		// Recursively collect dependencies from inner argument
		if arg.Inner != nil {
			r.collectDependencies(arg.Inner, deps)
		}
		// Note: TaggedArg is not handled here because it's already desugared to ServiceRefArg
		// by the time this phase runs.
	}
}

// Apply builds the dependency graph for all services
func (r *dependencyBuilderPhase) Apply(_ *di.Config, container *Container) error {
	for _, id := range xmaps.OrderedKeys(container.Services) {
		svc := container.Services[id]
		if svc.IsAlias() {
			svc.Dependencies = []*Service{svc.Alias}
			continue
		}
		if svc.Constructor == nil {
			continue
		}

		deps := make(map[string]*Service)

		// Method receiver is a dependency
		if svc.Constructor.Kind == MethodConstructor && svc.Constructor.Receiver != nil {
			deps[svc.Constructor.Receiver.ID] = svc.Constructor.Receiver
		}

		for _, arg := range svc.Constructor.Args {
			r.collectDependencies(arg, deps)
		}

		svc.Dependencies = make([]*Service, 0, len(deps))
		for _, dep := range deps {
			svc.Dependencies = append(svc.Dependencies, dep)
		}

		slices.SortFunc(svc.Dependencies, func(a, b *Service) int {
			return cmp.Compare(a.ID, b.ID)
		})
	}
	return nil
}
