package ir

import (
	"cmp"
	"fmt"
	"go/types"
	"slices"
	"strconv"

	di "github.com/asp24/gendi"
	"github.com/asp24/gendi/xmaps"
)

const (
	// TagServicePrefix is the prefix for desugared tag service IDs
	TagServicePrefix = "__tagged_with."
	// stdlibPkgPath is the package path for stdlib functions
	stdlibPkgPath = "github.com/asp24/gendi/stdlib"
)

// tagDesugarPhase transforms tags into regular services with MakeSlice constructors.
// After this phase, container.tags will be empty and all TaggedArg arguments
// will be converted to ServiceRefArg pointing to the desugared tag services.
type tagDesugarPhase struct {
	resolver TypeResolver
}

// desugar transforms all tags into services
func (p *tagDesugarPhase) desugar(_ *di.Config, container *Container) error {
	if len(container.tags) == 0 {
		return nil
	}

	// Process tags in deterministic order
	for _, tagName := range xmaps.OrderedKeys(container.tags) {
		tag := container.tags[tagName]

		if tag.ElementType == nil {
			// Skip tags without ElementType - they have no usages
			continue
		}

		// Create synthetic service for this tag
		svc, err := p.createTagService(tagName, tag)
		if err != nil {
			return fmt.Errorf("tag %q: %w", tagName, err)
		}

		container.Services[svc.ID] = svc
	}

	// Rewrite TaggedArg -> ServiceRefArg
	if err := p.rewriteTaggedArgs(container); err != nil {
		return err
	}

	// Clear tags - they are now services
	container.tags = make(map[string]*Tag)

	return nil
}

// createTagService creates a synthetic service for a tag
func (p *tagDesugarPhase) createTagService(tagName string, tag *Tag) (*Service, error) {
	serviceID := TagServicePrefix + tagName

	// Sort services by priority if needed
	services := p.sortTagServices(tag)

	// Create MakeSlice constructor
	constructor, err := p.createMakeSliceConstructor(tag.ElementType, services)
	if err != nil {
		return nil, err
	}

	sliceType := types.NewSlice(tag.ElementType)

	// Build dependencies from constructor args
	deps := make([]*Service, len(services))
	copy(deps, services)

	return &Service{
		ID:           serviceID,
		Type:         sliceType,
		Constructor:  constructor,
		Shared:       false, // Always create fresh slice
		Public:       tag.Public,
		Tags:         nil, // Desugared services don't have tags
		Dependencies: deps,
	}, nil
}

// sortTagServices sorts services according to tag's SortBy attribute
func (p *tagDesugarPhase) sortTagServices(tag *Tag) []*Service {
	if len(tag.Services) == 0 {
		return nil
	}

	sorted := slices.Clone(tag.Services)

	if tag.SortBy == "priority" {
		slices.SortFunc(sorted, func(a, b *Service) int {
			pa := getServiceTagPriority(a, tag.Name)
			pb := getServiceTagPriority(b, tag.Name)
			if pa != pb {
				return cmp.Compare(pb, pa) // Descending by priority
			}
			return cmp.Compare(a.ID, b.ID)
		})
	} else {
		slices.SortFunc(sorted, func(a, b *Service) int {
			return cmp.Compare(a.ID, b.ID)
		})
	}

	return sorted
}

// getServiceTagPriority extracts the priority attribute from a service's tag
func getServiceTagPriority(svc *Service, tagName string) int {
	for _, st := range svc.Tags {
		if st.Tag.Name != tagName {
			continue
		}
		if v, ok := st.Attributes["priority"]; ok {
			switch val := v.(type) {
			case int:
				return val
			case int64:
				return int(val)
			case float64:
				return int(val)
			case string:
				if parsed, err := strconv.Atoi(val); err == nil {
					return parsed
				}
			}
		}
	}
	return 0
}

// createMakeSliceConstructor creates a constructor using stdlib.MakeSlice
func (p *tagDesugarPhase) createMakeSliceConstructor(elemType types.Type, services []*Service) (*Constructor, error) {
	// Lookup stdlib.MakeSlice function
	fn, err := p.resolver.LookupFunc(stdlibPkgPath, "MakeSlice")
	if err != nil {
		return nil, fmt.Errorf("lookup stdlib.MakeSlice: %w", err)
	}

	// Format element type as string for instantiation
	elemTypeStr := types.TypeString(elemType, func(pkg *types.Package) string {
		return pkg.Path()
	})

	// Instantiate generic function
	sig, typeArgs, err := p.resolver.InstantiateFunc(fn, []string{elemTypeStr})
	if err != nil {
		return nil, fmt.Errorf("instantiate MakeSlice[%s]: %w", elemTypeStr, err)
	}

	// Build arguments - one ServiceRefArg per tagged service
	args := make([]*Argument, len(services))
	for i, svc := range services {
		args[i] = &Argument{
			Kind:    ServiceRefArg,
			Type:    svc.Type,
			Service: svc,
		}
	}

	// Get result type from signature
	resultType := sig.Results().At(0).Type()

	return &Constructor{
		Kind:         FuncConstructor,
		Func:         fn,
		TypeArgs:     typeArgs,
		Args:         args,
		Params:       []types.Type{resultType}, // variadic param is slice type
		ResultType:   resultType,
		ReturnsError: false,
		Variadic:     true,
	}, nil
}

// rewriteTaggedArgs rewrites all TaggedArg arguments to ServiceRefArg
// and updates service Dependencies accordingly
func (p *tagDesugarPhase) rewriteTaggedArgs(container *Container) error {
	for _, svc := range container.Services {
		if svc.Constructor == nil {
			continue
		}

		hasTaggedArgs := false
		for i, arg := range svc.Constructor.Args {
			if arg.Kind != TaggedArg {
				continue
			}
			hasTaggedArgs = true

			if arg.Tag == nil {
				return fmt.Errorf("service %q arg[%d]: TaggedArg has nil Tag", svc.ID, i)
			}

			tagServiceID := TagServicePrefix + arg.Tag.Name
			tagSvc, ok := container.Services[tagServiceID]
			if !ok {
				return fmt.Errorf("service %q arg[%d]: desugared tag service %q not found", svc.ID, i, tagServiceID)
			}

			// Rewrite to ServiceRefArg
			arg.Kind = ServiceRefArg
			arg.Service = tagSvc
			arg.Tag = nil
		}

		// Update Dependencies if service had TaggedArg
		if hasTaggedArgs {
			p.rebuildDependencies(svc)
		}
	}

	return nil
}

// rebuildDependencies rebuilds the Dependencies slice for a service
// based on its constructor arguments
func (p *tagDesugarPhase) rebuildDependencies(svc *Service) {
	if svc.Constructor == nil {
		return
	}

	deps := make(map[string]*Service)

	// Method receiver is a dependency
	if svc.Constructor.Kind == MethodConstructor && svc.Constructor.Receiver != nil {
		deps[svc.Constructor.Receiver.ID] = svc.Constructor.Receiver
	}

	// Collect dependencies from arguments
	for _, arg := range svc.Constructor.Args {
		if arg.Kind == ServiceRefArg && arg.Service != nil {
			deps[arg.Service.ID] = arg.Service
		}
	}

	// Build sorted slice
	svc.Dependencies = make([]*Service, 0, len(deps))
	for _, dep := range deps {
		svc.Dependencies = append(svc.Dependencies, dep)
	}

	slices.SortFunc(svc.Dependencies, func(a, b *Service) int {
		return cmp.Compare(a.ID, b.ID)
	})
}
