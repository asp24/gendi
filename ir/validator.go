package ir

import (
	"errors"
	"fmt"
	"strings"
)

// validator validates the IR for correctness
type validator struct{}

// validate runs all validation checks
func (v *validator) validate(ctx *buildContext) error {
	if err := v.validatePublicServices(ctx); err != nil {
		return err
	}
	if err := v.detectCycles(ctx); err != nil {
		return err
	}
	if err := v.detectDecoratorCycles(ctx); err != nil {
		return err
	}
	return nil
}

// validatePublicServices ensures at least one public service exists
func (v *validator) validatePublicServices(ctx *buildContext) error {
	for _, svc := range ctx.services {
		if svc.Public {
			return nil
		}
	}
	for _, tag := range ctx.tags {
		if tag.Public {
			return nil
		}
	}
	return errors.New("at least one public service or tag is required")
}

// detectCyclesDFS performs DFS-based cycle detection on a service graph.
// It accepts a neighbor function to traverse different types of relationships.
func (v *validator) detectCyclesDFS(
	services map[string]*Service,
	getNeighbors func(*Service) []*Service,
	errorPrefix string,
) error {
	visited := make(map[string]bool)
	stack := make(map[string]bool)

	var dfs func(svc *Service, path []string) error
	dfs = func(svc *Service, path []string) error {
		if svc == nil {
			return nil
		}
		if stack[svc.ID] {
			cycle := append(path, svc.ID)
			return fmt.Errorf("%s: %s", errorPrefix, strings.Join(cycle, " -> "))
		}
		if visited[svc.ID] {
			return nil
		}
		visited[svc.ID] = true
		stack[svc.ID] = true

		for _, neighbor := range getNeighbors(svc) {
			if err := dfs(neighbor, append(path, svc.ID)); err != nil {
				return err
			}
		}

		stack[svc.ID] = false
		return nil
	}

	for _, svc := range services {
		if err := dfs(svc, nil); err != nil {
			return err
		}
	}
	return nil
}

// detectCycles detects circular dependencies using DFS
func (v *validator) detectCycles(ctx *buildContext) error {
	return v.detectCyclesDFS(
		ctx.services,
		func(svc *Service) []*Service { return svc.Dependencies },
		"circular dependency",
	)
}

// detectDecoratorCycles detects circular decorator chains using DFS
func (v *validator) detectDecoratorCycles(ctx *buildContext) error {
	// Only check services that are decorators
	decorators := make(map[string]*Service)
	for id, svc := range ctx.services {
		if svc.Decorates != nil {
			decorators[id] = svc
		}
	}

	return v.detectCyclesDFS(
		decorators,
		func(svc *Service) []*Service {
			if svc.Decorates == nil {
				return nil
			}
			return []*Service{svc.Decorates}
		},
		"circular decorator chain",
	)
}
