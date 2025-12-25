package di

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/asp24/gendi/gomod"
)

// pathResolver resolves import paths to file paths.
type pathResolver interface {
	// CanResolve returns true if this resolver can handle the given import path.
	CanResolve(importPath string) bool
	// Resolve resolves the import path to a list of file paths.
	Resolve(baseDir, importPath string, fs fileSystem) ([]string, error)
}

// pathResolverComposite chains multiple path resolvers.
type pathResolverComposite struct {
	resolvers []pathResolver
}

// newPathResolverComposite creates a composite with the standard resolver chain.
func newPathResolverComposite() *pathResolverComposite {
	return &pathResolverComposite{
		resolvers: []pathResolver{
			&globResolver{},         // Try glob patterns first
			&absolutePathResolver{}, // Then absolute paths
			&localPathResolver{},    // Then local paths
			&modulePathResolver{},   // Finally module imports
		},
	}
}

func (c *pathResolverComposite) CanResolve(importPath string) bool {
	for _, resolver := range c.resolvers {
		if resolver.CanResolve(importPath) {
			return true
		}
	}

	return false
}

// Resolve attempts resolution with each resolver in the chain.
func (c *pathResolverComposite) Resolve(baseDir, importPath string, fs fileSystem) ([]string, error) {
	for _, resolver := range c.resolvers {
		if !resolver.CanResolve(importPath) {
			continue
		}
		results, err := resolver.Resolve(baseDir, importPath, fs)
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

// fileSystem abstracts file system operations for path resolvers.
type fileSystem struct {
	fileExists        func(path string) bool
	findModule        func(baseDir, importPath string) (moduleDir, modulePath, remainder string, err error)
	findDefaultConfig func(moduleDir string) (path string, ok bool)
	globFiles         func(pattern string) ([]string, error)
}

// globResolver handles glob patterns (*, ?, []).
type globResolver struct{}

func (r *globResolver) CanResolve(importPath string) bool {
	return strings.ContainsAny(importPath, "*?[")
}

func (r *globResolver) Resolve(baseDir, importPath string, fs fileSystem) ([]string, error) {
	// Absolute glob
	if filepath.IsAbs(importPath) {
		return fs.globFiles(importPath)
	}

	// Relative glob or non-module glob
	isExplicitRelative := strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../")
	if isExplicitRelative || !gomod.LooksLikeModulePath(importPath) {
		pattern := filepath.Join(baseDir, importPath)
		return fs.globFiles(pattern)
	}

	// Module glob
	return r.resolveModuleGlob(baseDir, importPath, fs)
}

func (r *globResolver) resolveModuleGlob(baseDir, importPath string, fs fileSystem) ([]string, error) {
	moduleDir, modulePath, remainder, err := fs.findModule(baseDir, importPath)
	if err != nil {
		return nil, err
	}
	if remainder == "" {
		path, ok := fs.findDefaultConfig(moduleDir)
		if !ok {
			return nil, fmt.Errorf("module %s has no gendi.yaml", modulePath)
		}
		return []string{path}, nil
	}
	pattern := filepath.Join(moduleDir, filepath.FromSlash(remainder))
	return fs.globFiles(pattern)
}

// absolutePathResolver handles absolute file paths.
type absolutePathResolver struct{}

func (r *absolutePathResolver) CanResolve(importPath string) bool {
	return filepath.IsAbs(importPath)
}

func (r *absolutePathResolver) Resolve(baseDir, importPath string, fs fileSystem) ([]string, error) {
	if !fs.fileExists(importPath) {
		return nil, fmt.Errorf("import not found at %s", importPath)
	}
	path, err := filepath.Abs(importPath)
	if err != nil {
		return nil, err
	}
	return []string{path}, nil
}

// localPathResolver handles local/relative file paths.
type localPathResolver struct{}

func (r *localPathResolver) CanResolve(importPath string) bool {
	// Local resolver tries to resolve any non-absolute, non-glob path
	// It checks if file exists locally
	return true // Always returns true; actual check happens in Resolve
}

func (r *localPathResolver) Resolve(baseDir, importPath string, fs fileSystem) ([]string, error) {
	localPath := filepath.Join(baseDir, importPath)
	if !fs.fileExists(localPath) {
		// If explicitly relative (./ or ../), fail immediately
		isExplicitRelative := strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../")
		if isExplicitRelative {
			return nil, fmt.Errorf("import not found at %s", localPath)
		}
		// Otherwise, let module resolver try
		return nil, nil
	}

	path, err := filepath.Abs(localPath)
	if err != nil {
		return nil, err
	}
	return []string{path}, nil
}

// modulePathResolver handles Go module imports.
type modulePathResolver struct{}

func (r *modulePathResolver) CanResolve(importPath string) bool {
	return gomod.LooksLikeModulePath(importPath)
}

func (r *modulePathResolver) Resolve(baseDir, importPath string, fs fileSystem) ([]string, error) {
	moduleDir, modulePath, remainder, err := fs.findModule(baseDir, importPath)
	if err != nil {
		return nil, err
	}

	if remainder == "" {
		// Looking for default config in module root
		if path, ok := fs.findDefaultConfig(moduleDir); ok {
			return []string{path}, nil
		}
		return nil, fmt.Errorf("module %s has no gendi.yaml", modulePath)
	}

	// Looking for specific file in module
	full := filepath.Join(moduleDir, filepath.FromSlash(remainder))
	if !fs.fileExists(full) {
		return nil, fmt.Errorf("module %s does not contain %s", modulePath, remainder)
	}

	path, err := filepath.Abs(full)
	if err != nil {
		return nil, err
	}
	return []string{path}, nil
}
