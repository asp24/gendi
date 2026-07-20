package yaml

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gendi-org/gendi/gomod"
)

// Confiner verifies that a config file's real location is within its declared
// boundary immediately before the file is loaded.
type Confiner struct{}

// Confine resolves boundary and path through symlinks and rejects a path whose
// real location is outside the real boundary. It returns path as an absolute
// addressed path, preserving its symlink spelling for relative imports and
// $this resolution.
func (Confiner) Confine(boundary, path string) (string, error) {
	if boundary == "" {
		return "", fmt.Errorf("boundary must not be empty")
	}

	absBoundary, err := filepath.Abs(boundary)
	if err != nil {
		return "", err
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	realBoundary, err := filepath.EvalSymlinks(absBoundary)
	if err != nil {
		return "", fmt.Errorf("resolve boundary %q: %w", absBoundary, err)
	}
	realPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		return "", fmt.Errorf("resolve config %q: %w", absPath, err)
	}

	rel, err := filepath.Rel(realBoundary, realPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("config %q resolves outside boundary %q", absPath, absBoundary)
	}
	return absPath, nil
}

// DefaultBoundary derives the load boundary for a root config file from its
// addressed location: the root of the containing Go module, or the config's
// own directory when it is not inside any module.
func DefaultBoundary(configPath string) (string, error) {
	abs, err := filepath.Abs(configPath)
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(abs)
	if root, _, found := gomod.FindModuleRoot(dir); found {
		return root, nil
	}
	return dir, nil
}
