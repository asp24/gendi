package imprt

import (
	"fmt"
	"path/filepath"
)

// ResolverAbs handles absolute file paths.
type ResolverAbs struct {
}

func (r *ResolverAbs) CanResolve(importPath string) bool {
	return filepath.IsAbs(importPath)
}

func (r *ResolverAbs) Resolve(_, importPath string) ([]string, error) {
	if !fileExists(importPath) {
		return nil, fmt.Errorf("import not found at %s", importPath)
	}
	path, err := filepath.Abs(importPath)
	if err != nil {
		return nil, err
	}
	return []string{path}, nil
}
