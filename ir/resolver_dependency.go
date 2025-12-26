package ir

import (
	"fmt"
	"go/types"
)

// dependencyResolver builds service dependency graph and links tagged services
type dependencyResolver struct{}

// resolve builds the dependency graph for all services
func (r *dependencyResolver) resolve(ctx *buildContext) error {
	// First, link services to their tags and validate types
	if err := r.resolveTaggedServices(ctx); err != nil {
		return err
	}

	// Then build dependency graph
	for _, svc := range ctx.services {
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
			case InnerArg:
				if svc.Decorates != nil {
					deps[svc.Decorates.ID] = svc.Decorates
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
	}
	return nil
}

// resolveTaggedServices links services to their tags and validates type compatibility
func (r *dependencyResolver) resolveTaggedServices(ctx *buildContext) error {
	for _, svc := range ctx.services {
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
