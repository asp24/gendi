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
	// InstantiateFunc instantiates a generic function with the given type arguments.
	// Returns the instantiated signature. typeArgs are type strings to resolve.
	InstantiateFunc(fn *types.Func, typeArgs []string) (*types.Signature, []types.Type, error)
}

// Builder constructs an IR Container from raw config.
type Builder struct {
	resolver TypeResolver
	cfg      *di.Config
}

// NewBuilder creates a new IR builder.
func NewBuilder(resolver TypeResolver, cfg *di.Config) *Builder {
	return &Builder{
		resolver: resolver,
		cfg:      cfg,
	}
}

// Build constructs the IR Container using a multi-phase approach.
func (b *Builder) Build() (*Container, error) {
	if b.cfg.Services == nil || len(b.cfg.Services) == 0 {
		return nil, errors.New("no services defined")
	}

	result := NewContainer()

	// Phase 1: Build foundational structures
	if err := (&parameterPhase{resolver: b.resolver}).build(b.cfg, result); err != nil {
		return nil, err
	}
	if err := (&tagPhase{resolver: b.resolver}).build(b.cfg, result); err != nil {
		return nil, err
	}
	if err := (&servicePhase{}).build(b.cfg, result); err != nil {
		return nil, err
	}

	// Phase 2: Resolve constructors and dependencies
	if err := (&constructorResolver{resolver: b.resolver}).resolve(b.cfg, result); err != nil {
		return nil, err
	}
	if err := (&decoratorResolver{}).resolve(b.cfg, result); err != nil {
		return nil, err
	}
	if err := (&dependencyResolver{}).resolve(b.cfg, result); err != nil {
		return nil, err
	}

	// Phase 3: Validate and analyze
	if err := (&validator{}).validate(b.cfg, result); err != nil {
		return nil, err
	}

	(&errorPropagator{}).propagate(b.cfg, result)

	// Phase 4: Optimizations
	pruneUnreachable(b.cfg, result)
	_ = (&sharedOptimizer{}).resolve(b.cfg, result)

	return result, nil
}
