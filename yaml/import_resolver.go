package yaml

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
	moduleLocator *gomod.Locator
}

// NewImportResolver creates a new import resolver.
func NewImportResolver() *ImportResolver {
	return &ImportResolver{}
}

// Resolve resolves an import path to a list of file paths.
// The baseDir is the directory containing the importing file.
func (r *ImportResolver) Resolve(baseDir, importPath string) ([]string, error) {
	if importPath == "" {
		return nil, fmt.Errorf("import path is empty")
	}
	if r.hasGlob(importPath) {
		return r.resolveGlob(baseDir, importPath)
	}
	if filepath.IsAbs(importPath) {
		path, err := r.ensureFile(importPath)
		if err != nil {
			return nil, err
		}
		return []string{path}, nil
	}
	localPath := filepath.Join(baseDir, importPath)
	if r.fileExists(localPath) {
		path, err := filepath.Abs(localPath)
		if err != nil {
			return nil, err
		}
		return []string{path}, nil
	}
	if r.isExplicitRelative(importPath) {
		return nil, fmt.Errorf("import not found at %s", localPath)
	}
	if !gomod.LooksLikeModulePath(importPath) {
		return nil, fmt.Errorf("import not found at %s", localPath)
	}
	path, err := r.resolveModuleImport(baseDir, importPath)
	if err != nil {
		return nil, err
	}
	return []string{path}, nil
}

func (r *ImportResolver) hasGlob(path string) bool {
	return strings.ContainsAny(path, "*?[")
}

func (r *ImportResolver) isExplicitRelative(path string) bool {
	return strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../")
}

func (r *ImportResolver) fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func (r *ImportResolver) ensureFile(path string) (string, error) {
	if r.fileExists(path) {
		return filepath.Abs(path)
	}
	return "", fmt.Errorf("import not found at %s", path)
}

func (r *ImportResolver) resolveGlob(baseDir, importPath string) ([]string, error) {
	if filepath.IsAbs(importPath) {
		return r.globFiles(importPath)
	}
	if r.isExplicitRelative(importPath) || !gomod.LooksLikeModulePath(importPath) {
		pattern := filepath.Join(baseDir, importPath)
		return r.globFiles(pattern)
	}
	return r.resolveModuleImportGlob(baseDir, importPath)
}

func (r *ImportResolver) resolveModuleImport(baseDir, importPath string) (string, error) {
	locator := gomod.NewLocator(baseDir)
	parts := strings.Split(importPath, "/")
	for i := len(parts); i >= 1; i-- {
		candidate := strings.Join(parts[:i], "/")
		if r.hasGlob(candidate) {
			continue
		}
		moduleDir, err := locator.FindModuleDir(candidate)
		if err != nil {
			continue
		}
		remainder := strings.Join(parts[i:], "/")
		if remainder == "" {
			if path, ok := r.findDefaultConfig(moduleDir); ok {
				return path, nil
			}
			return "", fmt.Errorf("module %s has no gendi.yaml", candidate)
		}
		full := filepath.Join(moduleDir, filepath.FromSlash(remainder))
		if r.fileExists(full) {
			return filepath.Abs(full)
		}
		return "", fmt.Errorf("module %s does not contain %s", candidate, remainder)
	}
	return "", fmt.Errorf("module %s not found", importPath)
}

func (r *ImportResolver) resolveModuleImportGlob(baseDir, importPath string) ([]string, error) {
	locator := gomod.NewLocator(baseDir)
	parts := strings.Split(importPath, "/")
	for i := len(parts); i >= 1; i-- {
		candidate := strings.Join(parts[:i], "/")
		if r.hasGlob(candidate) {
			continue
		}
		moduleDir, err := locator.FindModuleDir(candidate)
		if err != nil {
			continue
		}
		remainder := strings.Join(parts[i:], "/")
		if remainder == "" {
			path, ok := r.findDefaultConfig(moduleDir)
			if !ok {
				return nil, fmt.Errorf("module %s has no gendi.yaml", candidate)
			}
			return []string{path}, nil
		}
		pattern := filepath.Join(moduleDir, filepath.FromSlash(remainder))
		return r.globFiles(pattern)
	}
	return nil, fmt.Errorf("module %s not found", importPath)
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
