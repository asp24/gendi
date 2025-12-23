package generator

import (
	di "github.com/asp24/gendi"
)

// ErrorPropagationCalculator computes which service getters and builders can return errors.
type ErrorPropagationCalculator struct {
	cfg *di.Config
}

// NewErrorPropagationCalculator creates a new error propagation calculator.
func NewErrorPropagationCalculator(cfg *di.Config) *ErrorPropagationCalculator {
	return &ErrorPropagationCalculator{cfg: cfg}
}

// ErrorPropagationResult holds the computed error propagation maps.
type ErrorPropagationResult struct {
	BuildCanError  map[string]bool
	GetterCanError map[string]bool
}

// Calculate computes error propagation for the given services and decorator mappings.
func (c *ErrorPropagationCalculator) Calculate(services map[string]*serviceDef, decoratorsByBase map[string][]*serviceDef) ErrorPropagationResult {
	result := ErrorPropagationResult{
		BuildCanError:  make(map[string]bool),
		GetterCanError: make(map[string]bool),
	}

	// Initialize from constructors
	for id, svc := range services {
		result.GetterCanError[id] = svc.constructor.returnsError
		result.BuildCanError[id] = svc.constructor.returnsError
	}

	// Propagate errors until stable
	changed := true
	for changed {
		changed = false
		changed = c.propagateBuildErrors(services, &result) || changed
		changed = c.propagateGetterErrors(services, decoratorsByBase, &result) || changed
	}

	return result
}

func (c *ErrorPropagationCalculator) propagateBuildErrors(services map[string]*serviceDef, result *ErrorPropagationResult) bool {
	changed := false
	for id, svc := range services {
		can := svc.constructor.returnsError
		deps, _ := buildDeps(id, svc, c.cfg)
		for _, dep := range deps {
			if result.GetterCanError[dep] {
				can = true
			}
		}
		if result.BuildCanError[id] != can {
			result.BuildCanError[id] = can
			changed = true
		}
	}
	return changed
}

func (c *ErrorPropagationCalculator) propagateGetterErrors(services map[string]*serviceDef, decoratorsByBase map[string][]*serviceDef, result *ErrorPropagationResult) bool {
	changed := false
	for id, svc := range services {
		newVal := c.computeGetterCanError(id, svc, decoratorsByBase, result)
		if result.GetterCanError[id] != newVal {
			result.GetterCanError[id] = newVal
			changed = true
		}
	}
	return changed
}

func (c *ErrorPropagationCalculator) computeGetterCanError(id string, svc *serviceDef, decoratorsByBase map[string][]*serviceDef, result *ErrorPropagationResult) bool {
	if svc.isDecorator {
		return c.computeDecoratorGetterError(id, svc, decoratorsByBase, result)
	}
	if decs := decoratorsByBase[id]; len(decs) > 0 {
		return c.computeDecoratedServiceGetterError(id, decs, result)
	}
	return result.BuildCanError[id]
}

func (c *ErrorPropagationCalculator) computeDecoratorGetterError(id string, svc *serviceDef, decoratorsByBase map[string][]*serviceDef, result *ErrorPropagationResult) bool {
	base := svc.decorates
	if result.BuildCanError[base] {
		return true
	}
	decs := decoratorsByBase[base]
	for _, d := range decs {
		if result.BuildCanError[d.id] {
			return true
		}
		if d.id == id {
			break
		}
	}
	return false
}

func (c *ErrorPropagationCalculator) computeDecoratedServiceGetterError(id string, decs []*serviceDef, result *ErrorPropagationResult) bool {
	if result.BuildCanError[id] {
		return true
	}
	for _, d := range decs {
		if result.BuildCanError[d.id] {
			return true
		}
	}
	return false
}

// computeGetterErrors is a convenience function that wraps ErrorPropagationCalculator.
func computeGetterErrors(services map[string]*serviceDef, cfg *di.Config, decoratorsByBase map[string][]*serviceDef) (map[string]bool, map[string]bool) {
	calc := NewErrorPropagationCalculator(cfg)
	result := calc.Calculate(services, decoratorsByBase)
	return result.BuildCanError, result.GetterCanError
}
