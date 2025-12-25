package ir

import (
	"fmt"
	"go/types"

	di "github.com/asp24/gendi"
)

// argumentResolver resolves constructor arguments
type argumentResolver struct{}

// resolve resolves a single constructor argument
func (r *argumentResolver) resolve(ctx *buildContext, svcID string, idx int, arg di.Argument, paramType types.Type) (*Argument, error) {
	irArg := &Argument{Type: paramType}

	switch arg.Kind {
	case di.ArgServiceRef:
		dep, ok := ctx.services[arg.Value]
		if !ok {
			return nil, fmt.Errorf("service %q arg[%d]: unknown service %q", svcID, idx, arg.Value)
		}
		irArg.Kind = ServiceRefArg
		irArg.Service = dep

	case di.ArgInner:
		irArg.Kind = InnerArg
		irArg.Inner = true

	case di.ArgParam:
		param, ok := ctx.parameters[arg.Value]
		if !ok {
			// Parameter might be provided at runtime
			irArg.Kind = ParamRefArg
			irArg.Parameter = &Parameter{Name: arg.Value, Type: paramType}
		} else {
			if !types.AssignableTo(param.Type, paramType) {
				return nil, fmt.Errorf("service %q arg[%d]: parameter %q type mismatch", svcID, idx, arg.Value)
			}
			irArg.Kind = ParamRefArg
			irArg.Parameter = param
		}

	case di.ArgTagged:
		tag, ok := ctx.tags[arg.Value]
		if !ok {
			return nil, fmt.Errorf("service %q arg[%d]: unknown tag %q", svcID, idx, arg.Value)
		}
		irArg.Kind = TaggedArg
		irArg.Tag = tag

	default: // Literal
		litVal, err := convertLiteral(arg.Literal, paramType)
		if err != nil {
			return nil, fmt.Errorf("service %q arg[%d]: %w", svcID, idx, err)
		}
		irArg.Kind = LiteralArg
		irArg.Literal = litVal
	}

	return irArg, nil
}
