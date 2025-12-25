package ir

import (
	"errors"
	"go/types"

	di "github.com/asp24/gendi"
)

// TypeResolver resolves type strings to Go types.
type TypeResolver interface {
	LookupType(typeStr string) (types.Type, error)
	LookupFunc(pkgPath, name string) (*types.Func, error)
	LookupMethod(recv types.Type, name string) (*types.Func, error)
}

// Builder constructs an IR Container from raw config.
type Builder struct {
	cfg      *di.Config
	resolver TypeResolver
}

// NewBuilder creates a new IR builder.
func NewBuilder(cfg *di.Config, resolver TypeResolver) *Builder {
	return &Builder{
		cfg:      cfg,
		resolver: resolver,
	}
}

// Build constructs the IR Container using a multi-phase approach.
func (b *Builder) Build() (*Container, error) {
	if b.cfg.Services == nil || len(b.cfg.Services) == 0 {
		return nil, errors.New("no services defined")
	}

	ctx := newBuildContext(b.cfg, b.resolver)

	// Phase 1: Build foundational structures
	if err := (&parameterPhase{}).build(ctx); err != nil {
		return nil, err
	}
	if err := (&tagPhase{}).build(ctx); err != nil {
		return nil, err
	}
	if err := (&servicePhase{}).build(ctx); err != nil {
		return nil, err
	}

	// Phase 2: Resolve constructors and dependencies
	if err := (&constructorResolver{}).resolve(ctx); err != nil {
		return nil, err
	}
	if err := (&decoratorResolver{}).resolve(ctx); err != nil {
		return nil, err
	}
	if err := (&dependencyResolver{}).resolve(ctx); err != nil {
		return nil, err
	}

	// Phase 3: Validate and analyze
	if err := (&validator{}).validate(ctx); err != nil {
		return nil, err
	}
	(&errorPropagator{}).propagate(ctx)

	return &Container{
		Services:     ctx.services,
		Parameters:   ctx.parameters,
		Tags:         ctx.tags,
		ServiceOrder: ctx.order,
	}, nil
}

