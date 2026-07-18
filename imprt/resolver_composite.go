package imprt

import (
	"fmt"
	"path/filepath"
)

// ResolverComposite chains multiple path resolvers.
type ResolverComposite struct {
	resolvers []Resolver
}

func NewPathResolverComposite(resolvers ...Resolver) *ResolverComposite {
	return &ResolverComposite{
		resolvers: resolvers,
	}
}

// NewResolverCompositeDefault creates a composite with the standard resolver
// chain. boundaryRoot is the containment boundary for importing files that are
// not inside any Go module; within a module, each resolver confines to that
// module instead.
func NewResolverCompositeDefault(boundaryRoot string) *ResolverComposite {
	return NewPathResolverComposite(
		&ResolverGlob{boundaryRoot: boundaryRoot},  // Try glob patterns first
		&ResolverLocal{boundaryRoot: boundaryRoot}, // Then local paths
		&ResolverModule{},                          // Finally module imports
	)
}

func (c *ResolverComposite) CanResolve(importPath string) bool {
	for _, resolver := range c.resolvers {
		if resolver.CanResolve(importPath) {
			return true
		}
	}

	return false
}

// Resolve attempts resolution with each resolver in the chain. A resolver
// returning (nil, nil) is treated as "not mine" and the next one is tried;
// a non-nil error aborts the chain.
func (c *ResolverComposite) Resolve(baseDir, importPath string) ([]string, error) {
	if importPath == "" {
		return nil, fmt.Errorf("import path is empty")
	}
	if filepath.IsAbs(importPath) {
		return nil, fmt.Errorf("absolute paths are not allowed; use a path relative to the importing file or a Go module path")
	}

	for _, resolver := range c.resolvers {
		if !resolver.CanResolve(importPath) {
			continue
		}
		results, err := resolver.Resolve(baseDir, importPath)
		if err != nil {
			return nil, err
		}
		// LocalPathResolver returns nil if it can't resolve (to allow fallthrough)
		if results != nil {
			return results, nil
		}
	}
	// No resolver could handle this path
	localPath := filepath.Join(baseDir, importPath)
	return nil, fmt.Errorf("import not found at %s", localPath)
}
