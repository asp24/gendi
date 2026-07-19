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

// findModule locates a Go module by iterating through import path segments,
// resolving within the module context of the importing file: the module
// containing baseDir, or — when baseDir is outside any module — the module at
// the resolver's boundary. Without either, module imports are impossible and
// the error asks for an explicit project root. Successful lookups are
// memoized in moduleDirs per (context, modulePath). Returns (moduleDir,
// modulePath, remainder, error) where:
//   - moduleDir: absolute path to the module directory
//   - modulePath: the import path of the found module
//   - remainder: the path segment after the module path
func (r *Resolver) findModule(baseDir, importPath string) (string, string, string, error) {
	contextDir := pathToAbs(baseDir)
	if _, _, found := gomod.FindModuleRoot(contextDir); !found {
		if _, _, found := gomod.FindModuleRoot(r.boundary); !found {
			return "", "", "", fmt.Errorf("module import %q requires a Go module: no go.mod found above %s or the boundary %s — point the boundary at the project's module root", importPath, baseDir, r.boundary)
		}
		contextDir = r.boundary
	}

	locator := gomod.NewLocator(contextDir)
	parts := strings.Split(importPath, "/")
	for i := len(parts); i >= 1; i-- {
		candidate := strings.Join(parts[:i], "/")
		// Skip if candidate contains glob characters
		if strings.ContainsAny(candidate, "*?[") {
			continue
		}
		key := contextDir + "\x00" + candidate
		moduleDir, cached := r.moduleDirs[key]
		if !cached {
			var err error
			moduleDir, err = locator.FindModuleDir(candidate)
			if err != nil {
				continue
			}
			r.moduleDirs[key] = moduleDir
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
// boundary when dir is not inside any module. It is the containment boundary
// a path resolved relative to an importing file in dir may not escape.
func moduleRootOf(dir, boundary string) string {
	if root, _, found := gomod.FindModuleRoot(dir); found {
		return pathToAbs(root)
	}
	return pathToAbs(boundary)
}

// globMatches expands pattern and splits its matches into files and
// directories, absolute and sorted. A pattern that matches nothing —
// including one whose base directory does not exist — yields empty results
// without error; only a malformed pattern is an error.
func globMatches(pattern string) (files, dirs []string, err error) {
	matches, err := doublestar.FilepathGlob(pattern)
	if err != nil {
		return nil, nil, fmt.Errorf("glob %q: %w", pattern, err)
	}
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			// A match that cannot be stat'ed — typically a dangling
			// symlink — is skipped rather than failing the whole load.
			continue
		}
		abs, err := filepath.Abs(match)
		if err != nil {
			return nil, nil, err
		}
		if info.IsDir() {
			dirs = append(dirs, abs)
		} else {
			files = append(files, abs)
		}
	}
	sort.Strings(files)
	sort.Strings(dirs)
	return files, dirs, nil
}
