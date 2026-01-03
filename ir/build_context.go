package ir

import (
	di "github.com/asp24/gendi"
)

// buildContext holds shared state during IR construction
type buildContext struct {
	cfg *di.Config

	// Intermediate state - populated during build phases
	services   map[string]*Service
	parameters map[string]*Parameter
	tags       map[string]*Tag
	order      []string // Service IDs in sorted order
}

// newBuildContext creates a new build context
func newBuildContext(cfg *di.Config) *buildContext {
	return &buildContext{
		cfg:        cfg,
		services:   make(map[string]*Service),
		parameters: make(map[string]*Parameter),
		tags:       make(map[string]*Tag),
	}
}
