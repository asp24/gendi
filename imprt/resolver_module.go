package imprt

import (
	"fmt"
	"path/filepath"

	"github.com/asp24/gendi/gomod"
)

// ResolverModule handles Go module imports.
type ResolverModule struct {
}

func (r *ResolverModule) CanResolve(importPath string) bool {
	return gomod.LooksLikeModulePath(importPath)
}

func (r *ResolverModule) Resolve(baseDir, importPath string) ([]string, error) {
	moduleDir, modulePath, remainder, err := findModule(baseDir, importPath)
	if err != nil {
		return nil, err
	}

	if remainder == "" {
		// Looking for default config in module root
		if path, ok := findDefaultConfig(moduleDir); ok {
			return []string{path}, nil
		}
		return nil, fmt.Errorf("module %s has no gendi.yaml", modulePath)
	}

	// Looking for specific file in module
	full := filepath.Join(moduleDir, filepath.FromSlash(remainder))
	if !fileExists(full) {
		return nil, fmt.Errorf("module %s does not contain %s", modulePath, remainder)
	}

	path, err := filepath.Abs(full)
	if err != nil {
		return nil, err
	}
	return []string{path}, nil
}
