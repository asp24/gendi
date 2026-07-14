package imprt

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ResolverLocal handles local/relative file paths.
type ResolverLocal struct {
}

// CanResolve accepts any path shape: whether the file exists locally can only
// be determined in Resolve, which returns (nil, nil) to pass non-relative
// misses on to ResolverModule.
func (r *ResolverLocal) CanResolve(_ string) bool {
	return true
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
