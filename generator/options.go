package generator

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	Strict    bool // Default: true
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
// This consolidates all generator-specific "magic" in one place.
// Call this before passing Options to New().
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

	// 3. Resolve module info if needed
	if o.ModulePath == "" || o.ModuleRoot == "" {
		modInfo, err := ResolveModuleInfo()
		if err != nil {
			return fmt.Errorf("resolve module info: %w", err)
		}
		o.ModulePath = modInfo.Path
		o.ModuleRoot = modInfo.Root
	}

	// 4. Compute output path if needed
	if o.OutputPkgPath == "" {
		o.OutputPkgPath = ComputeOutputPkgPath(o.ModulePath, o.ModuleRoot, o.Out)
	}

	return nil
}
