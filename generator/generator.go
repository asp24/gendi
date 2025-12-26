package generator

import (
	"errors"
	"fmt"
	"go/format"

	di "github.com/asp24/gendi"
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

// Finalize resolves all auto-configuration and validates required fields.
// This consolidates all generator-specific "magic" in one place.
// Call this before passing Options to New().
func (o *Options) Finalize() error {
	// 1. Validate required fields
	if o.Out == "" {
		return errors.New("Out is required")
	}
	if o.Package == "" {
		return errors.New("Package is required")
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

type Generator struct {
	cfg     *di.Config
	passes  []di.Pass
	options Options
}

func New(cfg *di.Config, opts Options, passes []di.Pass) *Generator {
	return &Generator{cfg: cfg, passes: passes, options: opts}
}

// Generate produces the container code.
// Options must be finalized before calling New() (via Options.Finalize()).
func (g *Generator) Generate() ([]byte, error) {
	// Run compiler passes
	for _, pass := range g.passes {
		if err := pass.Process(g.cfg); err != nil {
			return nil, fmt.Errorf("compiler pass %q failed: %w", pass.Name(), err)
		}
	}

	// Build context
	ctx, err := g.buildContext()
	if err != nil {
		return nil, err
	}

	// Render code
	code, err := g.render(ctx)
	if err != nil {
		return nil, err
	}

	// Format
	formatted, err := format.Source(code)
	if err != nil {
		return nil, fmt.Errorf("format generated code: %w", err)
	}
	return formatted, nil
}

func (g *Generator) buildContext() (*genContext, error) {
	builder := NewContextBuilder(g.cfg, g.options)
	return builder.Build()
}
