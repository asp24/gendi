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
}

// NewCache creates a new package cache.
func NewCache(moduleRoot string) *Cache {
	return &Cache{
		packages:   make(map[string]*types.Package),
		moduleRoot: moduleRoot,
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

// Load loads packages by their import paths and caches them along with their dependencies.
func (c *Cache) Load(paths []string) error {
	if len(paths) == 0 {
		return nil
	}

	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedTypes |
			packages.NeedTypesInfo |
			packages.NeedSyntax |
			packages.NeedImports |
			packages.NeedDeps,
		Dir: c.moduleRoot,
	}

	pkgs, err := packages.Load(cfg, paths...)
	if err != nil {
		return fmt.Errorf("load packages: %w", err)
	}

	if err := collectPackageErrors(pkgs); err != nil {
		return err
	}

	seen := make(map[string]bool)
	for _, pkg := range pkgs {
		c.cacheTree(seen, pkg)
	}

	return nil
}

// cacheTree recursively caches a package and all its dependencies.
func (c *Cache) cacheTree(seen map[string]bool, pkg *packages.Package) {
	if pkg == nil {
		return
	}

	key := pkg.PkgPath
	if key == "" {
		key = pkg.ID
	}

	if key != "" {
		if seen[key] {
			return
		}
		seen[key] = true
	}

	if pkg.Types != nil && key != "" {
		c.packages[key] = pkg.Types
	}

	for _, imp := range pkg.Imports {
		c.cacheTree(seen, imp)
	}
}

// collectPackageErrors collects all errors from loaded packages.
func collectPackageErrors(pkgs []*packages.Package) error {
	var errs []string
	for _, pkg := range pkgs {
		for _, err := range pkg.Errors {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return errors.New(strings.Join(errs, "; "))
}
