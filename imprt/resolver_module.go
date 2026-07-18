package imprt

import (
	"fmt"
	"path/filepath"

	"github.com/gendi-org/gendi/gomod"
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
		return nil, fmt.Errorf("module import %q must reference a file, e.g. %s/gendi.yaml", importPath, modulePath)
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
	// A ".." in the remainder must not climb out of the resolved module.
	return confine(moduleDir, importPath, []string{path})
}
