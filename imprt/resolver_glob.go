package imprt

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gendi-org/gendi/gomod"
)

// ResolverGlob handles glob patterns (*, ?, []).
type ResolverGlob struct {
	// boundaryRoot bounds relative globs when the importing file is not inside
	// any Go module; within a module the boundary is that module's root.
	boundaryRoot string
}

func (r *ResolverGlob) CanResolve(importPath string) bool {
	return strings.ContainsAny(importPath, "*?[")
}

func (r *ResolverGlob) Resolve(baseDir, importPath string) ([]string, error) {
	// Relative glob or non-module glob
	isExplicitRelative := strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../")
	if isExplicitRelative || !gomod.LooksLikeModulePath(importPath) {
		pattern := filepath.Join(baseDir, importPath)
		files, err := globFiles(pattern)
		if err != nil {
			return nil, err
		}
		// A relative glob may not reach outside the importing file's module.
		return confine(moduleRootOf(baseDir, r.boundaryRoot), importPath, files)
	}

	// Module glob
	return r.resolveModuleGlob(baseDir, importPath)
}

func (r *ResolverGlob) resolveModuleGlob(baseDir, importPath string) ([]string, error) {
	moduleDir, modulePath, remainder, err := findModule(baseDir, importPath)
	if err != nil {
		return nil, err
	}
	if remainder == "" {
		path, ok := findDefaultConfig(moduleDir)
		if !ok {
			return nil, fmt.Errorf("module %s has no gendi.yaml", modulePath)
		}
		return []string{path}, nil
	}
	pattern := filepath.Join(moduleDir, filepath.FromSlash(remainder))

	files, err := globFiles(pattern)
	if err != nil {
		return nil, err
	}
	// A ".." in the remainder must not climb out of the resolved module.
	return confine(moduleDir, importPath, files)
}
