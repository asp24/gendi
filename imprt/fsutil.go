package imprt

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/gendi-org/gendi/gomod"
)

func fileExists(path string) bool {
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
func findModule(baseDir, importPath string) (string, string, string, error) {
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

func pathToAbs(path string) string {
	abs, err := filepath.Abs(path)
	if err == nil {
		return abs
	}

	return path
}

func findDefaultConfig(moduleDir string) (string, bool) {
	path := filepath.Join(moduleDir, "gendi.yaml")
	if fileExists(path) {
		return pathToAbs(path), true
	}

	path = filepath.Join(moduleDir, "gendi.yml")
	if fileExists(path) {
		return pathToAbs(path), true
	}
	return "", false
}

// globFiles expands a glob pattern to matching files and returns them
// together with the pattern's base directory (the fixed prefix before the
// first meta character).
func globFiles(pattern string) ([]string, string, error) {
	// An empty match set is a valid no-op, but a glob rooted at a
	// non-existent directory is almost certainly a typo.
	base, _ := doublestar.SplitPattern(filepath.ToSlash(pattern))
	baseDir := filepath.FromSlash(base)
	if info, err := os.Stat(baseDir); err != nil || !info.IsDir() {
		return nil, "", fmt.Errorf("import glob %q: directory %q does not exist", pattern, base)
	}

	matches, err := doublestar.FilepathGlob(pattern)
	if err != nil {
		return nil, "", err
	}
	files := make([]string, 0, len(matches))
	for _, match := range matches {
		if fileExists(match) {
			abs, err := filepath.Abs(match)
			if err != nil {
				return nil, "", err
			}
			files = append(files, abs)
		}
	}
	if len(files) != 0 {
		sort.Strings(files)
	}
	return files, pathToAbs(baseDir), nil
}
