package ir

import (
	"fmt"
	"go/types"

	di "github.com/asp24/gendi"
)

// argumentResolver resolves constructor arguments
type argumentResolver struct{}

// resolve resolves a single constructor argument
func (r *argumentResolver) resolve(container *Container, svcID string, idx int, arg di.Argument, paramType types.Type) (*Argument, error) {
	irArg := &Argument{Type: paramType}

	switch arg.Kind {
	case di.ArgServiceRef:
		dep, ok := container.Services[arg.Value]
		if !ok {
			return nil, fmt.Errorf("service %q arg[%d]: unknown service %q", svcID, idx, arg.Value)
		}
		irArg.Kind = ServiceRefArg
		irArg.Service = dep

	case di.ArgInner:
		return nil, fmt.Errorf("service %q arg[%d]: @.inner should have been resolved by DecoratorPass", svcID, idx)

	case di.ArgParam:
		param, ok := container.Parameters[arg.Value]
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
		tag, ok := container.tags[arg.Value]
		if !ok {
			// Create tag on-demand - infer ElementType from parameter
			tag = &Tag{
				Name:     arg.Value,
				Services: []*Service{},
			}
			container.tags[arg.Value] = tag
		}

		// Infer or validate ElementType from parameter type
		sliceType, ok := paramType.Underlying().(*types.Slice)
		if !ok {
			return nil, fmt.Errorf("service %q arg[%d]: tagged injection requires slice type, got %s", svcID, idx, paramType)
		}
		elemType := sliceType.Elem()

		if tag.ElementType == nil {
			// Infer ElementType from parameter
			tag.ElementType = elemType
		} else if !types.AssignableTo(tag.ElementType, elemType) {
			// Validate consistency if ElementType was declared or inferred earlier
			return nil, fmt.Errorf("service %q arg[%d]: tag %q element type mismatch: %s is not assignable to %s",
				svcID, idx, arg.Value, tag.ElementType, elemType)
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
