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

// moduleRootOf returns the absolute root of the Go module containing dir, or
// boundaryRoot when dir is not inside any module. It is the containment
// boundary a path resolved relative to an importing file in dir may not escape.
func moduleRootOf(dir, boundaryRoot string) string {
	if root, _, found := gomod.FindModuleRoot(dir); found {
		return pathToAbs(root)
	}
	return pathToAbs(boundaryRoot)
}

// within reports whether path is root itself or nested under it. Both are
// compared lexically, so callers must pass absolute, cleaned paths.
func within(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

// confine returns files unchanged when every entry is within root, otherwise an
// error naming the first file that escapes. root is made absolute first so a
// relative module dir still compares correctly against absolute resolved files.
func confine(root, importPath string, files []string) ([]string, error) {
	absRoot := pathToAbs(root)
	for _, file := range files {
		if !within(absRoot, file) {
			return nil, fmt.Errorf("import %q resolves outside %q: %s", importPath, absRoot, file)
		}
	}
	return files, nil
}

func globFiles(pattern string) ([]string, error) {
	// An empty match set is a valid no-op, but a glob rooted at a
	// non-existent directory is almost certainly a typo.
	base, _ := doublestar.SplitPattern(filepath.ToSlash(pattern))
	if info, err := os.Stat(filepath.FromSlash(base)); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("import glob %q: directory %q does not exist", pattern, base)
	}

	matches, err := doublestar.FilepathGlob(pattern)
	if err != nil {
		return nil, err
	}
	files := make([]string, 0, len(matches))
	for _, match := range matches {
		if fileExists(match) {
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
