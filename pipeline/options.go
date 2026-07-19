package pipeline

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gendi-org/gendi/gomod"
	"github.com/gendi-org/gendi/typeres"
)

type Options struct {
	// Output (required)
	Out     string
	Package string

	// Optional - auto-resolved if empty
	Container     string // Default: "Container"
	ModulePath    string // Auto: from go.mod
	ModuleRoot    string // Auto: from go.mod
	OutputPkgPath string // Auto: computed from Out

	// Optional
	BuildTags string
	Verbose   bool
}

func (o *Options) computeOutput(out string) (string, error) {
	if strings.HasSuffix(out, ".go") {
		return out, nil
	}
	info, err := os.Stat(out)
	if err == nil && info.IsDir() {
		return filepath.Join(out, "container_gen.go"), nil
	}
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}

	// Treat as directory when it doesn't exist and doesn't end with .go.
	return filepath.Join(out, "container_gen.go"), nil
}

// Finalize resolves all autoconfiguration and validates required fields.
// This consolidates all pipeline-specific "magic" in one place.
// Call this before passing Options to Emit().
func (o *Options) Finalize() error {
	// 1. Validate required fields
	if o.Out == "" {
		return errors.New("out is required")
	}

	var err error
	if o.Out, err = o.computeOutput(o.Out); err != nil {
		return fmt.Errorf("output path: %w", err)
	}

	if o.Package == "" {
		return errors.New("package is required")
	}

	// 2. Set defaults
	if o.Container == "" {
		o.Container = "Container"
	}

	// 3. Resolve module info from the output location if needed: the
	// generated file's package identity is defined by the module that will
	// contain it, never by the process working directory.
	if o.ModulePath == "" || o.ModuleRoot == "" {
		absOut, err := filepath.Abs(o.Out)
		if err != nil {
			return fmt.Errorf("resolve module info: %w", err)
		}
		outDir := filepath.Dir(absOut)
		root, modPath, found := gomod.FindModuleRoot(outDir)
		if !found {
			return fmt.Errorf("resolve module info: go.mod not found above output directory %s", outDir)
		}
		o.ModulePath = modPath
		o.ModuleRoot = root
	}

	// 4. Compute output path if needed
	if o.OutputPkgPath == "" {
		pkgPath, err := typeres.ComputeOutputPkgPath(o.ModulePath, o.ModuleRoot, o.Out)
		if err != nil {
			return fmt.Errorf("compute output package path: %w", err)
		}
		o.OutputPkgPath = pkgPath
	}

	return nil
}
