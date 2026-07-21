package imprt

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
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

// globMatches expands the slash-separated glob pattern relative to root and
// returns the matched files as absolute sorted paths. Glob syntax is
// interpreted only in pattern — a metacharacter in root is a literal path
// byte. An existing base directory that yields no matches is a valid no-op,
// but a base directory that does not exist is almost always a typo and is a
// generation-time error; a malformed pattern is likewise an error.
func globMatches(root, pattern string) ([]string, error) {
	base, glob := doublestar.SplitPattern(path.Clean(filepath.ToSlash(pattern)))
	dir := filepath.Join(root, filepath.FromSlash(base))
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("import glob %q: directory %q does not exist", pattern, base)
	}
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
	return strings.ContainsAny(pattern, "*?[{")
}
