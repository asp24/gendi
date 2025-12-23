package generator

import (
	"fmt"
	"strings"

	di "github.com/asp24/gendi"
)

// CycleDetector detects circular dependencies in the service graph using DFS.
type CycleDetector struct {
	services map[string]*serviceDef
	cfg      *di.Config
	visited  map[string]bool
	stack    map[string]bool
}

// NewCycleDetector creates a new cycle detector for the given services.
func NewCycleDetector(services map[string]*serviceDef, cfg *di.Config) *CycleDetector {
	return &CycleDetector{
		services: services,
		cfg:      cfg,
		visited:  make(map[string]bool),
		stack:    make(map[string]bool),
	}
}

// Detect checks for circular dependencies and returns an error if found.
func (d *CycleDetector) Detect() error {
	for id := range d.services {
		if err := d.dfs(id, nil); err != nil {
			return err
		}
	}
	return nil
}

func (d *CycleDetector) dfs(id string, path []string) error {
	if d.stack[id] {
		cycle := append(path, id)
		return fmt.Errorf("circular dependency: %s", strings.Join(cycle, " -> "))
	}
	if d.visited[id] {
		return nil
	}
	d.visited[id] = true
	d.stack[id] = true

	deps, err := constructorDeps(id, d.services[id], d.cfg)
	if err != nil {
		return err
	}
	for _, dep := range deps {
		if err := d.dfs(dep, append(path, id)); err != nil {
			return err
		}
	}

	d.stack[id] = false
	return nil
}
