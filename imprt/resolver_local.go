package imprt

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ResolverLocal handles local/relative file paths.
type ResolverLocal struct {
}

func (r *ResolverLocal) CanResolve(_ string) bool {
	// Local resolver tries to resolve any non-absolute, non-glob path
	// It checks if file exists locally
	return true // Always returns true; actual check happens in Resolve
}

func (r *ResolverLocal) Resolve(baseDir, importPath string) ([]string, error) {
	localPath := filepath.Join(baseDir, importPath)
	if !fileExists(localPath) {
		// If explicitly relative (./ or ../), fail immediately
		isExplicitRelative := strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../")
		if isExplicitRelative {
			return nil, fmt.Errorf("import not found at %s", localPath)
		}
		// Otherwise, let module resolver try
		return nil, nil
	}

	path, err := filepath.Abs(localPath)
	if err != nil {
		return nil, err
	}

	return []string{path}, nil
}
