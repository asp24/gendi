package imprt

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ResolverLocal handles local/relative file paths.
type ResolverLocal struct {
	// boundaryRoot bounds resolution when the importing file is not inside any
	// Go module; within a module the boundary is that module's root.
	boundaryRoot string
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

	// A relative import may not escape the module of the importing file (a
	// cleaned "../" chain, or a dotted first segment that looks module-shaped).
	return confine(moduleRootOf(baseDir, r.boundaryRoot), importPath, []string{path})
}
