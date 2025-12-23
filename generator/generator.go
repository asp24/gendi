package generator

import (
	"fmt"
	"go/format"

	di "github.com/asp24/gendi"
)

type Options struct {
	Out           string
	Package       string
	Container     string
	Strict        bool
	BuildTags     string
	Verbose       bool
	ModulePath    string
	ModuleRoot    string
	OutputPkgPath string
}

type Generator struct {
	cfg     *di.Config
	passes  []di.Pass
	options Options
}

func New(cfg *di.Config, opts Options, passes []di.Pass) *Generator {
	return &Generator{cfg: cfg, passes: passes, options: opts}
}

func (g *Generator) Generate() ([]byte, error) {
	if g.options.Strict {
		// strict is default; keep here for clarity
	}
	if g.options.Container == "" {
		g.options.Container = "Container"
	}

	for _, pass := range g.passes {
		if err := pass.Process(g.cfg); err != nil {
			return nil, fmt.Errorf("compiler pass %q failed: %w", pass.Name(), err)
		}
	}

	ctx, err := g.buildContext()
	if err != nil {
		return nil, err
	}

	code, err := g.render(ctx)
	if err != nil {
		return nil, err
	}

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
