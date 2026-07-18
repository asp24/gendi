package imprt

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gendi-org/gendi/gomod"
)

// ResolverSandbox confines file resolution to the Go module that contains the
// importing file. It wraps another resolver and enforces two rules:
//
//   - An absolute import path is rejected outright: everything inside the
//     module is addressable relative to the importing file, and other modules
//     are addressed by module path.
//   - Any file that resolves outside the importing file's module root is
//     rejected. Reaching another module is only possible through a module-path
//     import, which is bounded by the go.mod graph.
//
// The boundary is the module of the importing file, not the top-level module,
// so a dependency's own config may reference its own sibling files while still
// being unable to reach into the consumer's filesystem.
type ResolverSandbox struct {
	inner        Resolver
	fallbackRoot string
}

// NewResolverSandbox wraps inner, confining resolution to the module of each
// importing file. fallbackRoot is the boundary used for configs that are not
// inside any Go module (no go.mod found walking up from the importing file).
func NewResolverSandbox(inner Resolver, fallbackRoot string) *ResolverSandbox {
	return &ResolverSandbox{inner: inner, fallbackRoot: pathToAbs(fallbackRoot)}
}

func (s *ResolverSandbox) CanResolve(importPath string) bool {
	return s.inner.CanResolve(importPath)
}

func (s *ResolverSandbox) Resolve(baseDir, importPath string) ([]string, error) {
	if filepath.IsAbs(importPath) {
		return nil, fmt.Errorf("absolute paths are not allowed; use a path relative to the importing file or a Go module path")
	}

	files, err := s.inner.Resolve(baseDir, importPath)
	if err != nil {
		return nil, err
	}

	// A file outside the importing module is only legitimate when it was
	// reached through a module-path import, which is bounded by the go.mod
	// graph. Everything else must stay within the module root.
	root := s.moduleRoot(baseDir)
	moduleImport := s.isModulePathImport(importPath)
	for _, file := range files {
		if s.within(root, file) || moduleImport {
			continue
		}
		return nil, fmt.Errorf("import %q resolves outside module root %q: %s", importPath, root, file)
	}
	return files, nil
}

// isModulePathImport reports whether importPath is a Go module path, using the
// same shape rules as the underlying resolver chain: an explicitly relative
// path is never a module path, regardless of dots in its segments.
func (s *ResolverSandbox) isModulePathImport(importPath string) bool {
	if strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../") {
		return false
	}
	return gomod.LooksLikeModulePath(importPath)
}

// moduleRoot returns the root of the Go module containing dir, or the
// configured fallback root when dir is not inside any module.
func (s *ResolverSandbox) moduleRoot(dir string) string {
	if root, _, found := gomod.FindModuleRoot(dir); found {
		return pathToAbs(root)
	}
	return s.fallbackRoot
}

// within reports whether path is root itself or nested under it.
func (s *ResolverSandbox) within(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
