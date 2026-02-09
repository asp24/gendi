package generator

import (
	di "github.com/asp24/gendi"
	"github.com/asp24/gendi/ir"
	"github.com/asp24/gendi/typeres"
)

// EmitOptions holds rendering-specific options for code generation.
type EmitOptions struct {
	Package       string
	Container     string
	BuildTags     string
	OutputPkgPath string
}

// Emit converts an IR container into formatted Go source code.
func Emit(cfg *di.Config, irContainer *ir.Container, typeResolver *typeres.Resolver, opts EmitOptions) ([]byte, error) {
	irConverter := NewIRConverter(typeResolver)
	ctx, err := irConverter.convert(irContainer, cfg)
	if err != nil {
		return nil, err
	}

	e := newEmitter(opts)
	return e.emit(cfg, ctx)
}
