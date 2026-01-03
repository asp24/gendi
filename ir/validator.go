package ir

import (
	"errors"
	"fmt"
	"strings"

	di "github.com/asp24/gendi"
)

// validator validates the IR for correctness
type validator struct{}

// validate runs all validation checks
func (v *validator) validate(_ *di.Config, container *Container) error {
	if err := v.validatePublicServices(container); err != nil {
		return err
	}
	if err := v.detectCycles(container); err != nil {
		return err
	}
	return nil
}

// validatePublicServices ensures at least one public service exists
func (v *validator) validatePublicServices(container *Container) error {
	for _, svc := range container.Services {
		if svc.Public {
			return nil
		}
	}
	for _, tag := range container.Tags {
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
func (v *validator) detectCycles(container *Container) error {
	return v.detectCyclesDFS(
		container.Services,
		func(svc *Service) []*Service { return svc.Dependencies },
		"circular dependency",
	)
}
