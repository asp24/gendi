package di

import (
	"fmt"
	"sort"
)

// DecoratorPass transforms decorator services into plain services and aliases.
// This is an internal mandatory pass that runs before user passes.
//
// Transformation example:
//   Input:
//     base: { constructor: ... }
//     dec: { decorates: "base", args: ["@.inner"] }
//   Output:
//     dec.inner: { constructor: <original base constructor> }
//     dec: { args: ["@dec.inner"] }
//     base: { alias: "dec" }
type DecoratorPass struct{}

func (p *DecoratorPass) Name() string {
	return "decorator"
}

type decoratorPassState struct {
	decoratorToBase     map[string]string
	baseToDecorators    map[string][]string
	decoratorToPriority map[string]int
}

func (s *decoratorPassState) popNext() (baseID string, decoratorID string, ok bool) {
	for base, decorators := range s.baseToDecorators {
		decoratorID = decorators[0]
		delete(s.decoratorToBase, decoratorID)

		decorators = decorators[1:]
		s.baseToDecorators[base] = decorators
		if len(decorators) == 0 {
			delete(s.baseToDecorators, base)
		}

		return base, decoratorID, true
	}

	return "", "", false
}

func (p *DecoratorPass) Process(cfg *Config) (*Config, error) {
	state, err := p.buildState(cfg)
	if err != nil {
		return nil, err
	}

	for {
		baseID, decoratorID, ok := state.popNext()
		if !ok {
			break
		}

		if err := p.expandOne(cfg, baseID, decoratorID); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

func (p *DecoratorPass) buildState(cfg *Config) (*decoratorPassState, error) {
	state := &decoratorPassState{
		decoratorToBase:     make(map[string]string),
		baseToDecorators:    make(map[string][]string),
		decoratorToPriority: make(map[string]int),
	}

	for id, svc := range cfg.Services {
		if svc.Decorates == "" {
			continue
		}

		base, ok := cfg.Services[svc.Decorates]
		if !ok {
			return nil, fmt.Errorf("decorator %q decorates unknown service %q", id, svc.Decorates)
		}
		if base.Decorates != "" {
			return nil, fmt.Errorf("decorator %q cannot be decorated", svc.Decorates)
		}

		state.decoratorToBase[id] = svc.Decorates
		state.baseToDecorators[svc.Decorates] = append(state.baseToDecorators[svc.Decorates], id)
		state.decoratorToPriority[id] = svc.DecorationPriority
	}

	// Sort decorators by priority (lower priority first, stable sort by ID)
	for baseID, decs := range state.baseToDecorators {
		if len(decs) <= 1 {
			continue
		}
		sort.SliceStable(decs, func(i, j int) bool {
			return state.decoratorToPriority[decs[i]] < state.decoratorToPriority[decs[j]]
		})
		state.baseToDecorators[baseID] = decs
	}

	return state, nil
}

func (p *DecoratorPass) expandOne(cfg *Config, baseID, decoratorID string) error {
	baseSvc := cfg.Services[baseID]
	decoratorSvc := cfg.Services[decoratorID]

	// Determine inner service (reuse existing alias or create new)
	innerID, innerSvc := p.resolveInnerService(cfg, baseSvc, decoratorID)

	// Transform base into alias to current decorator
	aliasService := Service{
		Alias:  decoratorID,
		Shared: copyBoolPtr(baseSvc.Shared),
		Public: baseSvc.Public, // Preserve public flag
	}

	// Rewrite @.inner args in decorator
	decoratorSvc.Constructor.Args = p.rewriteInnerArgs(decoratorSvc.Constructor.Args, innerID)
	decoratorSvc.Decorates = "" // Clear decoration marker

	// Propagate shared flag (only to decorator and alias, NOT inner)
	p.propagateShared(&decoratorSvc, &aliasService)

	// Update config (need to write back all modified services)
	cfg.Services[innerID] = innerSvc
	cfg.Services[decoratorID] = decoratorSvc
	cfg.Services[baseID] = aliasService

	return nil
}

func (p *DecoratorPass) resolveInnerService(cfg *Config, baseSvc Service, decoratorID string) (string, Service) {
	// If base is already an alias, reuse the target instead of creating new inner
	if baseSvc.Alias != "" {
		innerID := baseSvc.Alias
		return innerID, cfg.Services[innerID]
	}

	// Create new inner service (clone of base)
	innerID := decoratorID + ".inner"
	innerSvc := Service{
		Type:        baseSvc.Type,
		Constructor: baseSvc.Constructor,
		Shared:      copyBoolPtr(baseSvc.Shared),
		Public:      false, // Inner services are never public
		Alias:       "",
		Decorates:   "",
		Tags:        nil, // No tags on inner
	}
	// Store the new inner service
	cfg.Services[innerID] = innerSvc
	return innerID, innerSvc
}

func (p *DecoratorPass) rewriteInnerArgs(args []Argument, innerServiceID string) []Argument {
	result := make([]Argument, len(args))
	for i, arg := range args {
		if arg.Kind != ArgInner {
			result[i] = arg
			continue
		}
		result[i] = Argument{
			Kind:  ArgServiceRef,
			Value: innerServiceID,
		}
	}
	return result
}

func (p *DecoratorPass) propagateShared(services ...*Service) {
	// Collect explicit shared values (bool OR checks like IR logic)
	var explicitShared *bool
	for _, svc := range services {
		if svc.Shared == nil {
			continue
		}
		if explicitShared == nil {
			// First explicit value
			val := *svc.Shared
			explicitShared = &val
			continue
		}
		// OR with existing value
		if *svc.Shared {
			sharedTrue := true
			explicitShared = &sharedTrue
		}
	}

	// If no explicit shared value, leave all as is (default behavior from IR)
	if explicitShared == nil {
		return
	}

	// Propagate explicit shared value to all services
	for _, svc := range services {
		svc.Shared = copyBoolPtr(explicitShared)
	}
}

func copyBoolPtr(p *bool) *bool {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}
