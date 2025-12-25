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

// NewResolverCompositeDefault creates a composite with the standard resolver chain.
func NewResolverCompositeDefault() *ResolverComposite {
	fs := fileSystem{}
	return NewPathResolverComposite(
		&ResolverGlob{fs: fs},   // Try glob patterns first
		&ResolverAbs{fs: fs},    // Then absolute paths
		&ResolverLocal{fs: fs},  // Then local paths
		&ResolverModule{fs: fs}, // Finally module imports
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

// Resolve attempts resolution with each resolver in the chain.
func (c *ResolverComposite) Resolve(baseDir, importPath string) ([]string, error) {
	if importPath == "" {
		return nil, fmt.Errorf("import path is empty")
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
