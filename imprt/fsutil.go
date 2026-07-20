package imprt

import (
	"fmt"
	"os"
	"path"
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
// the error asks for an explicit project root. Lookups are memoized in
// moduleDirs per (context, modulePath), failures included. Returns
// (moduleDir, modulePath, remainder, error) where:
//   - moduleDir: absolute path to the module directory
//   - modulePath: the import path of the found module
//   - remainder: the path segment after the module path
func (r *Resolver) findModule(baseDir, importPath string) (string, string, string, error) {
	contextDir := baseDir
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
		lookup, cached := r.moduleDirs[key]
		if !cached {
			dir, err := locator.FindModuleDir(candidate)
			lookup = moduleLookup{dir: dir, ok: err == nil}
			r.moduleDirs[key] = lookup
		}
		if !lookup.ok {
			continue
		}
		remainder := strings.Join(parts[i:], "/")
		return lookup.dir, candidate, remainder, nil
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

// globMatches expands the slash-separated glob pattern relative to root and
// returns the matched files as absolute sorted paths. Glob syntax is
// interpreted only in pattern — a metacharacter in root is a literal path
// byte. A pattern that matches nothing — including one whose base directory
// does not exist — yields no results without error; only a malformed pattern
// is an error.
func globMatches(root, pattern string) ([]string, error) {
	base, glob := doublestar.SplitPattern(path.Clean(pattern))
	dir := filepath.Join(root, filepath.FromSlash(base))
	matches, err := doublestar.Glob(os.DirFS(dir), glob)
	if err != nil {
		return nil, fmt.Errorf("glob %q: %w", pattern, err)
	}
	var files []string
	for _, match := range matches {
		full := filepath.Join(dir, match)
		info, err := os.Stat(full)
		if err != nil {
			// A match that cannot be stat'ed — typically a dangling
			// symlink — is skipped rather than failing the whole load.
			continue
		}
		if !info.IsDir() {
			files = append(files, full)
		}
	}
	sort.Strings(files)
	return files, nil
}
