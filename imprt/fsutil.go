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

func isGlobPattern(pattern string) bool {
	return strings.ContainsAny(pattern, "*?[")
}

// localMatch reports whether pattern names an existing file (or, for a glob,
// at least one match) relative to baseDir — the signal that a module-shaped
// spelling was really meant as a local path. Best-effort: a malformed glob
// is treated as no match, deferring the authoritative error to resolution.
func localMatch(baseDir, pattern string) bool {
	if isGlobPattern(pattern) {
		files, err := globMatches(baseDir, pattern)
		return err == nil && len(files) > 0
	}
	return fileExists(filepath.Join(baseDir, pattern))
}
