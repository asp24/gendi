package yaml

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gendi-org/gendi/gomod"
)

// Confiner verifies that a config file's real location is within its declared
// boundary immediately before the file is loaded. It memoizes each boundary's
// symlink-resolved root and Go-module identity: the same boundary recurs for
// every candidate loaded within it, and resolving it costs a symlink
// evaluation plus a go.mod walk. A Confiner is single-threaded, like the loader
// that owns it, so the cache needs no locking.
type Confiner struct {
	boundaries map[string]resolvedBoundary
}

// NewConfiner creates a Confiner with an empty boundary cache.
func NewConfiner() *Confiner {
	return &Confiner{boundaries: map[string]resolvedBoundary{}}
}

// resolvedBoundary is a boundary resolved through symlinks together with the Go
// module that contains it, computed once and reused across candidates.
type resolvedBoundary struct {
	real       string
	moduleRoot string
	modulePath string
	hasModule  bool
}

// Confine resolves boundary and path through symlinks and rejects a path whose
// real location is outside the real boundary. It returns path as an absolute
// addressed path — preserving its symlink spelling for relative imports and
// $this resolution — together with its symlink-resolved real path, the
// canonical identity the caller uses for cycle detection.
func (c *Confiner) Confine(boundary, path string) (abs, real string, err error) {
	if boundary == "" {
		return "", "", fmt.Errorf("boundary must not be empty")
	}

	b, err := c.resolveBoundary(boundary)
	if err != nil {
		return "", "", err
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", "", err
	}
	realPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		return "", "", fmt.Errorf("resolve config %q: %w", absPath, err)
	}

	rel, err := filepath.Rel(b.real, realPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("config %q resolves outside boundary %q", absPath, b.real)
	}

	configModuleRoot, configModulePath, configHasModule := gomod.FindModuleRoot(filepath.Dir(realPath))

	switch {
	case b.hasModule && !configHasModule:
		return "", "", fmt.Errorf(
			"config %q leaves Go module %s at %q",
			absPath,
			b.modulePath,
			b.moduleRoot,
		)
	case !b.hasModule && configHasModule:
		return "", "", fmt.Errorf(
			"config %q enters Go module %s at %q from a non-module boundary; use a module-path import",
			absPath,
			configModulePath,
			configModuleRoot,
		)
	case b.hasModule &&
		configHasModule &&
		filepath.Clean(b.moduleRoot) != filepath.Clean(configModuleRoot):
		return "", "", fmt.Errorf(
			"config %q crosses Go module boundary from %s at %q to %s at %q; use a module-path import",
			absPath,
			b.modulePath,
			b.moduleRoot,
			configModulePath,
			configModuleRoot,
		)
	}

	return absPath, realPath, nil
}

// resolveBoundary resolves boundary through symlinks and identifies the Go
// module containing it, caching the result so a repeated boundary costs only a
// map lookup.
func (c *Confiner) resolveBoundary(boundary string) (resolvedBoundary, error) {
	absBoundary, err := filepath.Abs(boundary)
	if err != nil {
		return resolvedBoundary{}, err
	}
	if b, ok := c.boundaries[absBoundary]; ok {
		return b, nil
	}

	realBoundary, err := filepath.EvalSymlinks(absBoundary)
	if err != nil {
		return resolvedBoundary{}, fmt.Errorf("resolve boundary %q: %w", absBoundary, err)
	}
	moduleRoot, modulePath, hasModule := gomod.FindModuleRoot(realBoundary)
	b := resolvedBoundary{
		real:       realBoundary,
		moduleRoot: moduleRoot,
		modulePath: modulePath,
		hasModule:  hasModule,
	}
	c.boundaries[absBoundary] = b
	return b, nil
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
