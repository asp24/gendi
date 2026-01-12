package ir

import (
	"fmt"
	"go/types"

	di "github.com/asp24/gendi"
	"github.com/asp24/gendi/yaml"
)

// argResolver resolves constructor arguments
type argResolver struct{}

// resolve resolves a single constructor argument
func (r *argResolver) resolve(container *Container, svcID string, idx int, arg di.Argument, paramType types.Type) (*Argument, error) {
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

	case di.ArgSpread:
		// Validate that parameter is variadic (represented as slice in go/types)
		if _, ok := paramType.Underlying().(*types.Slice); !ok {
			return nil, fmt.Errorf("service %q arg[%d]: !spread: can only be used with variadic parameters, got %s", svcID, idx, paramType)
		}

		// Parse inner argument string
		innerKind, innerValue := yaml.ParseArgumentString(arg.Value)
		innerArg := di.Argument{
			Kind:  innerKind,
			Value: innerValue,
		}
		// Preserve literal only if inner is also a literal
		if innerKind == di.ArgLiteral {
			innerArg.Literal = arg.Literal
		}

		// Resolve inner argument with the slice type (not element type)
		// The inner expression should evaluate to []T
		innerResolved, err := r.resolve(container, svcID, idx, innerArg, paramType)
		if err != nil {
			return nil, fmt.Errorf("service %q arg[%d]: !spread: %w", svcID, idx, err)
		}

		// Validate that inner resolved to a slice type
		if _, ok := innerResolved.Type.Underlying().(*types.Slice); !ok {
			return nil, fmt.Errorf("service %q arg[%d]: !spread: requires slice type, got %s", svcID, idx, innerResolved.Type)
		}

		irArg.Kind = SpreadArg
		irArg.Inner = innerResolved

	default: // Literal
		// Validate that this is actually a literal argument
		if arg.Kind != di.ArgLiteral {
			return nil, fmt.Errorf("service %q arg[%d]: unknown argument kind %d", svcID, idx, arg.Kind)
		}
		litVal, err := convertLiteral(arg.Literal, paramType)
		if err != nil {
			return nil, fmt.Errorf("service %q arg[%d]: %w", svcID, idx, err)
		}
		irArg.Kind = LiteralArg
		irArg.Literal = litVal
	}

	return irArg, nil
}
