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
	return errors.New("at least one public service is required")
}

// detectCycles detects circular dependencies using DFS
func (v *validator) detectCycles(ctx *buildContext) error {
	visited := make(map[string]bool)
	stack := make(map[string]bool)

	var dfs func(svc *Service, path []string) error
	dfs = func(svc *Service, path []string) error {
		if stack[svc.ID] {
			cycle := append(path, svc.ID)
			return fmt.Errorf("circular dependency: %s", strings.Join(cycle, " -> "))
		}
		if visited[svc.ID] {
			return nil
		}
		visited[svc.ID] = true
		stack[svc.ID] = true

		for _, dep := range svc.Dependencies {
			if err := dfs(dep, append(path, svc.ID)); err != nil {
				return err
			}
		}

		stack[svc.ID] = false
		return nil
	}

	for _, svc := range ctx.services {
		if err := dfs(svc, nil); err != nil {
			return err
		}
	}
	return nil
}

// detectDecoratorCycles detects circular decorator chains using DFS
func (v *validator) detectDecoratorCycles(ctx *buildContext) error {
	visited := make(map[string]bool)
	stack := make(map[string]bool)

	var dfs func(svc *Service, path []string) error
	dfs = func(svc *Service, path []string) error {
		if svc == nil || svc.Decorates == nil {
			return nil
		}
		if stack[svc.ID] {
			cycle := append(path, svc.ID)
			return fmt.Errorf("circular decorator chain: %s", strings.Join(cycle, " -> "))
		}
		if visited[svc.ID] {
			return nil
		}
		visited[svc.ID] = true
		stack[svc.ID] = true

		// Follow the decorator chain through Decorates link
		if err := dfs(svc.Decorates, append(path, svc.ID)); err != nil {
			return err
		}

		stack[svc.ID] = false
		return nil
	}

	for _, svc := range ctx.services {
		if svc.Decorates != nil {
			if err := dfs(svc, nil); err != nil {
				return err
			}
		}
	}
	return nil
}
