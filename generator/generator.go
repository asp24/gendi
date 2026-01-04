package generator

import (
	"fmt"
	"go/format"

	di "github.com/asp24/gendi"
)

type Generator struct {
	cfg     *di.Config
	options Options
}

func New(cfg *di.Config, opts Options) *Generator {
	return &Generator{cfg: cfg, options: opts}
}

// Generate produces the container code.
// Options must be finalized before calling New() (via Options.Finalize()).
// Config should already have passes applied (via di.ApplyPasses()).
func (g *Generator) Generate() ([]byte, error) {
	// Build context
	ctx, rnd, err := g.buildContext()
	if err != nil {
		return nil, err
	}

	// Render code
	code, err := g.render(ctx, rnd)
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

func (g *Generator) buildContext() (*genContext, *Renderer, error) {
	builder := NewContextBuilder(g.cfg, g.options)
	return builder.Build()
}
