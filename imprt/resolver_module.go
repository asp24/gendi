package imprt

import (
	"fmt"
	"path/filepath"

	"github.com/asp24/gendi/gomod"
)

// ResolverModule handles Go module imports.
type ResolverModule struct {
	fs fileSystem
}

func (r *ResolverModule) CanResolve(importPath string) bool {
	return gomod.LooksLikeModulePath(importPath)
}

func (r *ResolverModule) Resolve(baseDir, importPath string) ([]string, error) {
	moduleDir, modulePath, remainder, err := r.fs.findModule(baseDir, importPath)
	if err != nil {
		return nil, err
	}

	if remainder == "" {
		// Looking for default config in module root
		if path, ok := r.fs.findDefaultConfig(moduleDir); ok {
			return []string{path}, nil
		}
		return nil, fmt.Errorf("module %s has no gendi.yaml", modulePath)
	}

	// Looking for specific file in module
	full := filepath.Join(moduleDir, filepath.FromSlash(remainder))
	if !r.fs.fileExists(full) {
		return nil, fmt.Errorf("module %s does not contain %s", modulePath, remainder)
	}

	path, err := filepath.Abs(full)
	if err != nil {
		return nil, err
	}
	return []string{path}, nil
}
