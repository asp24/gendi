package imprt

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gendi-org/gendi/gomod"
)

// ResolverGlob handles glob patterns (*, ?, []).
type ResolverGlob struct {
}

func (r *ResolverGlob) CanResolve(importPath string) bool {
	return strings.ContainsAny(importPath, "*?[")
}

func (r *ResolverGlob) Resolve(baseDir, importPath string) (*Resolution, error) {
	// Absolute glob: resolved files live under the glob base, not the
	// importing file's directory.
	if filepath.IsAbs(importPath) {
		files, globBase, err := globFiles(importPath)
		if err != nil {
			return nil, err
		}
		return &Resolution{Files: files, BaseDir: globBase}, nil
	}

	// Relative glob or non-module glob: resolved relative to the importing
	// file's directory.
	isExplicitRelative := strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../")
	if isExplicitRelative || !gomod.LooksLikeModulePath(importPath) {
		pattern := filepath.Join(baseDir, importPath)
		files, _, err := globFiles(pattern)
		if err != nil {
			return nil, err
		}
		return &Resolution{Files: files, BaseDir: baseDir}, nil
	}

	// Module glob
	return r.resolveModuleGlob(baseDir, importPath)
}

func (r *ResolverGlob) resolveModuleGlob(baseDir, importPath string) (*Resolution, error) {
	moduleDir, modulePath, remainder, err := findModule(baseDir, importPath)
	if err != nil {
		return nil, err
	}
	if remainder == "" {
		path, ok := findDefaultConfig(moduleDir)
		if !ok {
			return nil, fmt.Errorf("module %s has no gendi.yaml", modulePath)
		}
		return &Resolution{Files: []string{path}, BaseDir: pathToAbs(moduleDir)}, nil
	}
	pattern := filepath.Join(moduleDir, filepath.FromSlash(remainder))

	files, _, err := globFiles(pattern)
	if err != nil {
		return nil, err
	}
	return &Resolution{Files: files, BaseDir: pathToAbs(moduleDir)}, nil
}
