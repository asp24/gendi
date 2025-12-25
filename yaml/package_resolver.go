package yaml

import (
	"fmt"
	"path/filepath"

	"github.com/asp24/gendi/gomod"
)

// resolvePackagePath determines the Go package path for a given directory.
// It uses the gomod package to find the module root and computes the
// full package path relative to that root.
func resolvePackagePath(dir string) (string, error) {
	// Make path absolute
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Find go.mod using existing gomod package
	modRoot, modPath, found := gomod.FindModuleRoot(absDir)
	if !found {
		return "", fmt.Errorf("no go.mod found in %s or any parent directory", absDir)
	}

	// Get relative path from module root to directory
	relPath, err := filepath.Rel(modRoot, absDir)
	if err != nil {
		return "", fmt.Errorf("failed to compute relative path: %w", err)
	}

	// If directory is module root, return module path
	if relPath == "." {
		return modPath, nil
	}

	// Convert filesystem path to Go package path (use forward slashes)
	relPath = filepath.ToSlash(relPath)

	// Construct full package path
	return modPath + "/" + relPath, nil
}
