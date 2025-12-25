package imprt

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/asp24/gendi/gomod"
)

// ResolverGlob handles glob patterns (*, ?, []).
type ResolverGlob struct {
}

func (r *ResolverGlob) CanResolve(importPath string) bool {
	return strings.ContainsAny(importPath, "*?[")
}

func (r *ResolverGlob) Resolve(baseDir, importPath string) ([]string, error) {
	// Absolute glob
	if filepath.IsAbs(importPath) {
		return globFiles(importPath)
	}

	// Relative glob or non-module glob
	isExplicitRelative := strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../")
	if isExplicitRelative || !gomod.LooksLikeModulePath(importPath) {
		pattern := filepath.Join(baseDir, importPath)
		return globFiles(pattern)
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

	return globFiles(pattern)
}
