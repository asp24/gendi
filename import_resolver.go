package di

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/asp24/gendi/gomod"
)

// ImportResolver resolves import paths to actual file paths.
// It handles absolute paths, relative paths, glob patterns, and Go module imports.
type ImportResolver struct {
	pathResolver pathResolver
}

// NewImportResolver creates a new import resolver.
func NewImportResolver() *ImportResolver {
	return &ImportResolver{
		pathResolver: newPathResolverComposite(),
	}
}

// Resolve resolves an import path to a list of file paths.
// The baseDir is the directory containing the importing file.
func (r *ImportResolver) Resolve(baseDir, importPath string) ([]string, error) {
	if importPath == "" {
		return nil, fmt.Errorf("import path is empty")
	}

	// Create file system abstraction
	fs := fileSystem{
		fileExists:        r.fileExists,
		findModule:        r.findModule,
		findDefaultConfig: r.findDefaultConfig,
		globFiles:         r.globFiles,
	}

	return r.pathResolver.Resolve(baseDir, importPath, fs)
}

func (r *ImportResolver) fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// findModule locates a Go module by iterating through import path segments.
// Returns (moduleDir, modulePath, remainder, error) where:
//   - moduleDir: absolute path to the module directory
//   - modulePath: the import path of the found module
//   - remainder: the path segment after the module path
func (r *ImportResolver) findModule(baseDir, importPath string) (string, string, string, error) {
	locator := gomod.NewLocator(baseDir)
	parts := strings.Split(importPath, "/")
	for i := len(parts); i >= 1; i-- {
		candidate := strings.Join(parts[:i], "/")
		// Skip if candidate contains glob characters
		if strings.ContainsAny(candidate, "*?[") {
			continue
		}
		moduleDir, err := locator.FindModuleDir(candidate)
		if err != nil {
			continue
		}
		remainder := strings.Join(parts[i:], "/")
		return moduleDir, candidate, remainder, nil
	}
	return "", "", "", fmt.Errorf("module %s not found", importPath)
}

func (r *ImportResolver) findDefaultConfig(moduleDir string) (string, bool) {
	path := filepath.Join(moduleDir, "gendi.yaml")
	if r.fileExists(path) {
		abs, err := filepath.Abs(path)
		if err == nil {
			return abs, true
		}
		return path, true
	}
	path = filepath.Join(moduleDir, "gendi.yml")
	if r.fileExists(path) {
		abs, err := filepath.Abs(path)
		if err == nil {
			return abs, true
		}
		return path, true
	}
	return "", false
}

func (r *ImportResolver) globFiles(pattern string) ([]string, error) {
	matches, err := doublestar.FilepathGlob(pattern)
	if err != nil {
		return nil, err
	}
	files := make([]string, 0, len(matches))
	for _, match := range matches {
		if r.fileExists(match) {
			abs, err := filepath.Abs(match)
			if err != nil {
				return nil, err
			}
			files = append(files, abs)
		}
	}
	if len(files) != 0 {
		sort.Strings(files)
	}
	return files, nil
}
