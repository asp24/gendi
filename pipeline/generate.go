package pipeline

import (
	di "github.com/asp24/gendi"
	"github.com/asp24/gendi/generator"
)

// Generate compiles a DI config and emits generated Go source code.
// Options must be finalized before calling Generate (via Options.Finalize()).
func Generate(cfg *di.Config, opts Options) ([]byte, error) {
	compiled, err := Build(cfg, opts.ModuleRoot)
	if err != nil {
		return nil, err
	}

	return generator.Emit(compiled.Config, compiled.IR, compiled.TypeResolver, generator.EmitOptions{
		Package:       opts.Package,
		Container:     opts.Container,
		BuildTags:     opts.BuildTags,
		OutputPkgPath: opts.OutputPkgPath,
	})
}
