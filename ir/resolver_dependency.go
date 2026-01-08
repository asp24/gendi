package ir

import (
	"cmp"
	"fmt"
	"go/types"
	"slices"

	di "github.com/asp24/gendi"
	"github.com/asp24/gendi/xmaps"
)

// dependencyResolver builds service dependency graph and links tagged services
type dependencyResolver struct{}

// resolve builds the dependency graph for all services
func (r *dependencyResolver) resolve(_ *di.Config, container *Container) error {
	// First, link services to their tags and validate types
	if err := r.resolveTaggedServices(container); err != nil {
		return err
	}

	// Then build dependency graph
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
			switch arg.Kind {
			case ServiceRefArg:
				if arg.Service != nil {
					deps[arg.Service.ID] = arg.Service
				}
			case TaggedArg:
				if arg.Tag != nil {
					for _, tagged := range arg.Tag.Services {
						deps[tagged.ID] = tagged
					}
				}
			}
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

// resolveTaggedServices links services to their tags and validates type compatibility
func (r *dependencyResolver) resolveTaggedServices(container *Container) error {
	for _, id := range xmaps.OrderedKeys(container.Services) {
		svc := container.Services[id]
		for _, st := range svc.Tags {
			// Validate service type is assignable to tag's ElementType (if known)
			if st.Tag.ElementType != nil && svc.Type != nil {
				if !types.AssignableTo(svc.Type, st.Tag.ElementType) {
					return fmt.Errorf("service %q with tag %q: type %s is not assignable to %s",
						svc.ID, st.Tag.Name, svc.Type, st.Tag.ElementType)
				}
			}
			st.Tag.Services = append(st.Tag.Services, svc)
		}
	}
	return nil
}
