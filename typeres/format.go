package typeres

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ComputeOutputPkgPath calculates the Go package import path for the output file.
// It computes this based on the module path, module root, and output file location.
// An output directory outside the module root has no valid import path and is
// reported as an error.
func ComputeOutputPkgPath(modPath, modRoot, outFile string) (string, error) {
	outDir := outFile
	if strings.HasSuffix(outDir, ".go") {
		outDir = filepath.Dir(outDir)
	}

	outDir, err := filepath.Abs(outDir)
	if err != nil {
		return "", fmt.Errorf("resolve output directory %q: %w", outFile, err)
	}
	rel, err := filepath.Rel(modRoot, outDir)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("output directory %q is outside module root %q", outDir, modRoot)
	}

	rel = filepath.ToSlash(rel)
	if rel == "." {
		return modPath, nil
	}

	return modPath + "/" + rel, nil
}
