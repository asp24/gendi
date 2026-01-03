package typeres

import (
	"go/types"
	"path/filepath"
	"strings"
)

// ComputeOutputPkgPath calculates the Go package import path for the output file.
// It computes this based on the module path, module root, and output file location.
func ComputeOutputPkgPath(modPath, modRoot, outFile string) string {
	outDir := outFile
	if strings.HasSuffix(outDir, ".go") {
		outDir = filepath.Dir(outDir)
	}

	outDir, _ = filepath.Abs(outDir)
	rel, err := filepath.Rel(modRoot, outDir)
	if err != nil {
		return modPath
	}

	rel = filepath.ToSlash(rel)
	if rel == "." {
		return modPath
	}

	return modPath + "/" + rel
}

// FormatTypeString formats a types.Type as a string, using short package names
// when the package matches the output package path.
func FormatTypeString(t types.Type, outputPkgPath string) string {
	return types.TypeString(t, func(pkg *types.Package) string {
		if pkg.Path() == outputPkgPath {
			return ""
		}
		return pkg.Name()
	})
}
