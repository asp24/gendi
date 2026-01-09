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

	// Transform tags into services
	if err := r.transformTagsToServices(container); err != nil {
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

func (r *dependencyResolver) transformTagsToServices(container *Container) error {
	for _, name := range xmaps.OrderedKeys(container.Tags) {
		tag := container.Tags[name]

		// Sort services by priority
		// We can't access Attributes here directly from slice of Services.
		// But resolveTaggedServices populated tag.Services.
		// We lookup priority from the service's Tags list again.
		slices.SortFunc(tag.Services, func(a, b *Service) int {
			pa := r.getTagPriority(a, name)
			pb := r.getTagPriority(b, name)
			if pa == pb {
				return cmp.Compare(a.ID, b.ID)
			}
			// Higher priority first
			return cmp.Compare(pb, pa)
		})

		// Create synthetic service for the tag
		// We use "tagged_with_" prefix.
		// NOTE: Users might have IDs starting with this, but it's unlikely to collide 
		// if we assume standard naming.
		id := "tagged_with_" + name
		if _, exists := container.Services[id]; exists {
			return fmt.Errorf("generated tag service ID %q conflicts with existing service", id)
		}

		sliceType := types.NewSlice(tag.ElementType)
		
		args := make([]*Argument, len(tag.Services))
		for i, svc := range tag.Services {
			args[i] = &Argument{
				Kind:    ServiceRefArg,
				Type:    tag.ElementType,
				Service: svc,
			}
		}

		tagSvc := &Service{
			ID:     id,
			Type:   sliceType,
			Shared: false, // Always create new slice (elements are refs)
			Public: tag.Public,
			Constructor: &Constructor{
				Kind:       SliceConstructor,
				ResultType: sliceType,
				Args:       args,
			},
		}

		container.Services[id] = tagSvc

		// Replace usages in other services
		for _, svc := range container.Services {
			if svc.Constructor == nil {
				continue
			}
			for _, arg := range svc.Constructor.Args {
				if arg.Kind == TaggedArg && arg.Tag == tag {
					arg.Kind = ServiceRefArg
					arg.Service = tagSvc
					arg.Tag = nil
				}
			}
		}
	}

	// Clear tags map as they are now services
	container.Tags = make(map[string]*Tag)

	return nil
}

func (r *dependencyResolver) getTagPriority(svc *Service, tagName string) int {
	for _, st := range svc.Tags {
		if st.Tag.Name == tagName {
			if v, ok := st.Attributes["priority"]; ok {
				switch val := v.(type) {
				case int:
					return val
				case int64:
					return int(val)
				case float64:
					return int(val)
				}
			}
		}
	}
	return 0
}
