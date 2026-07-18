package typeres

import (
	"errors"
	"fmt"
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"
)

// Cache handles loading and caching of Go packages.
type Cache struct {
	packages   map[string]*types.Package
	moduleRoot string
	buildTags  string
}

// NewCache creates a new package cache. buildTags is passed to the package
// loader as-is via -tags= when non-empty.
func NewCache(moduleRoot, buildTags string) *Cache {
	return &Cache{
		packages:   make(map[string]*types.Package),
		moduleRoot: moduleRoot,
		buildTags:  buildTags,
	}
}

// Get retrieves a package from the cache.
// Returns an error if the package has not been loaded.
func (c *Cache) Get(path string) (*types.Package, error) {
	if pkg, ok := c.packages[path]; ok {
		return pkg, nil
	}
	return nil, fmt.Errorf("package %q not loaded", path)
}

// Load loads packages by their import paths and caches their type information.
func (c *Cache) Load(paths []string) error {
	return c.LoadWithCandidates(paths, nil)
}

// LoadWithCandidates loads required paths together with candidate paths
// derived from ambiguous qualified names (e.g. field access on a Go symbol,
// where the package/symbol boundary is unknown). Candidates that do not
// resolve to a real package are skipped instead of failing the load. A single
// packages.Load call is used so all results share one type universe.
func (c *Cache) LoadWithCandidates(required, candidates []string) error {
	if len(required)+len(candidates) == 0 {
		return nil
	}

	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedTypes,
		Dir: c.moduleRoot,
	}
	if c.buildTags != "" {
		cfg.BuildFlags = []string{"-tags=" + c.buildTags}
	}

	candidateSet := make(map[string]bool, len(candidates))
	for _, p := range candidates {
		candidateSet[p] = true
	}

	pkgs, err := packages.Load(cfg, append(append([]string{}, required...), candidates...)...)
	if err != nil {
		return fmt.Errorf("load packages: %w", err)
	}

	var errs []string
	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			if candidateSet[pkg.PkgPath] || candidateSet[pkg.ID] {
				continue
			}
			for _, pkgErr := range pkg.Errors {
				errs = append(errs, pkgErr.Error())
			}
			continue
		}
		c.cachePackage(pkg)
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}

	return nil
}

// cachePackage caches a loaded package's type information. Imports are not
// walked: the load mode omits NeedImports, and every package the resolver
// needs is requested explicitly by the caller.
func (c *Cache) cachePackage(pkg *packages.Package) {
	key := pkg.PkgPath
	if key == "" {
		key = pkg.ID
	}

	if key != "" && pkg.Types != nil {
		c.packages[key] = pkg.Types
	}
}
