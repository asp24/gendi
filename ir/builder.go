package ir

import (
	"errors"
	"fmt"
	"go/types"

	di "github.com/asp24/gendi"
)

// TypeResolver resolves type strings to Go types.
type TypeResolver interface {
	LookupType(typeStr string) (types.Type, error)
	LookupFunc(pkgPath, name string) (*types.Func, error)
	LookupMethod(recv types.Type, name string) (*types.Func, error)
	LookupVar(pkgPath, name string) (types.Object, error)
	// InstantiateFunc instantiates a generic function with the given type arguments.
	// Returns the instantiated signature. typeArgs are type strings to resolve.
	InstantiateFunc(fn *types.Func, typeArgs []string) (*types.Signature, []types.Type, error)
}

// Builder constructs an IR Container from raw config.
type Builder struct {
	phases []Phase
}

// NewBuilder creates a new IR builder.
func NewBuilder(resolver TypeResolver) *Builder {
	phases := []Phase{
		// Phase 1: Build foundational structures
		&parameterPhase{resolver: resolver},
		&tagPhase{resolver: resolver},
		&servicePhase{},

		// Phase 2: Resolve constructors
		&constructorResolverPhase{typeResolver: resolver, argResolver: &argResolver{typeResolver: resolver}},
		&autoTagPhase{},
		// Desugar tags into synthetic services (links services to tags, creates tag services, rewrites args)
		&tagDesugarPhase{resolver: resolver},
		// Build dependency graph for all services (requires desugared tags)
		&dependencyBuilderPhase{},

		// Phase 3: Validate and analyze
		&validatorPhase{},

		// Phase 4: Optimizations
		&unreachablePrunePhase{},
		&unusedParamPrunePhase{},
		&sharedOptimizerPhase{},
	}
	return &Builder{
		phases: phases,
	}
}

// Build constructs the IR Container using a multi-phase approach.
func (b *Builder) Build(cfg *di.Config) (*Container, error) {
	if cfg.Services == nil || len(cfg.Services) == 0 {
		return nil, errors.New("no services defined")
	}

	result := NewContainer()

	for _, phase := range b.phases {
		if err := phase.Apply(cfg, result); err != nil {
			return nil, fmt.Errorf("phase %T apply: %w", phase, err)
		}
	}

	return result, nil
}
