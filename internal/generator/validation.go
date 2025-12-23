package generator

import (
	"fmt"
	"go/types"

	"gopkg.in/yaml.v3"

	di "github.com/asp24/gendi"
)

func validateConstructorSignature(sig *types.Signature) (types.Type, bool, error) {
	res := sig.Results()
	if res.Len() == 0 || res.Len() > 2 {
		return nil, false, fmt.Errorf("constructor must return T or (T, error)")
	}
	resType := res.At(0).Type()
	returnsErr := false
	if res.Len() == 2 {
		errType := res.At(1).Type()
		if !isErrorType(errType) {
			return nil, false, fmt.Errorf("second return value must be error")
		}
		returnsErr = true
	}
	if err := validateServiceType(resType); err != nil {
		return nil, false, err
	}
	return resType, returnsErr, nil
}

func validateServiceType(t types.Type) error {
	switch tt := t.(type) {
	case *types.Pointer:
		return validateServiceType(tt.Elem())
	case *types.Named:
		return nil
	case *types.TypeParam:
		return fmt.Errorf("service type must not be type parameter")
	default:
		return fmt.Errorf("service type must be a named concrete type")
	}
}

func isErrorType(t types.Type) bool {
	return types.Identical(t, types.Universe.Lookup("error").Type())
}

func validateArgs(id string, svc *serviceDef, services map[string]*serviceDef, cfg *di.Config, loader *typeLoader, decoratorsByBase map[string][]*serviceDef, canError map[string]bool) error {
	cons := svc.constructor
	params := cons.params
	if len(cons.argDefs) != len(params) {
		return fmt.Errorf("service %q constructor args count mismatch: expected %d got %d", id, len(params), len(cons.argDefs))
	}
	for i, arg := range cons.argDefs {
		paramType := params[i]
		switch arg.Kind {
		case di.ArgServiceRef:
			dep := services[arg.Value]
			if dep == nil {
				return fmt.Errorf("service %q arg[%d]: unknown service %q", id, i, arg.Value)
			}
			depType := getterType(dep, services, decoratorsByBase)
			if !types.AssignableTo(depType, paramType) {
				return fmt.Errorf("service %q arg[%d]: expected %s, got %s", id, i, loader.typeString(paramType), loader.typeString(depType))
			}
		case di.ArgInner:
			if svc.decorates == "" {
				return fmt.Errorf("service %q arg[%d]: @.inner used outside decorator", id, i)
			}
			baseSvc := services[svc.decorates]
			if baseSvc == nil {
				return fmt.Errorf("service %q arg[%d]: unknown base service %q", id, i, svc.decorates)
			}
			innerType := baseSvc.typeName
			if baseSvc.cfg.Type != "" {
				declType, err := loader.lookupType(baseSvc.cfg.Type)
				if err != nil {
					return fmt.Errorf("service %q arg[%d]: base %q type: %w", id, i, svc.decorates, err)
				}
				innerType = declType
			}
			if !types.AssignableTo(innerType, paramType) {
				return fmt.Errorf("service %q arg[%d]: expected %s, got %s", id, i, loader.typeString(paramType), loader.typeString(innerType))
			}
		case di.ArgParam:
			param, ok := cfg.Parameters[arg.Value]
			if ok {
				if param.Type == "" {
					return fmt.Errorf("service %q arg[%d]: parameter %q missing type", id, i, arg.Value)
				}
				paramDefType, err := loader.lookupType(param.Type)
				if err != nil {
					return fmt.Errorf("service %q arg[%d]: parameter %q type: %w", id, i, arg.Value, err)
				}
				if _, err := paramGetterMethod(paramDefType); err != nil {
					return fmt.Errorf("service %q arg[%d]: parameter %q type: %w", id, i, arg.Value, err)
				}
				if !types.AssignableTo(paramDefType, paramType) {
					return fmt.Errorf("service %q arg[%d]: parameter %q expected %s, got %s", id, i, arg.Value, loader.typeString(paramType), loader.typeString(paramDefType))
				}
				break
			}
			if _, err := paramGetterMethod(paramType); err != nil {
				return fmt.Errorf("service %q arg[%d]: parameter %q type: %w", id, i, arg.Value, err)
			}
		case di.ArgTagged:
			tag, ok := cfg.Tags[arg.Value]
			if !ok {
				return fmt.Errorf("service %q arg[%d]: unknown tag %q", id, i, arg.Value)
			}
			if tag.ElementType == "" {
				return fmt.Errorf("service %q arg[%d]: tag %q missing element_type", id, i, arg.Value)
			}
			elemType, err := loader.lookupType(tag.ElementType)
			if err != nil {
				return fmt.Errorf("service %q arg[%d]: tag %q element_type: %w", id, i, arg.Value, err)
			}
			sliceType := types.NewSlice(elemType)
			if !types.AssignableTo(sliceType, paramType) {
				return fmt.Errorf("service %q arg[%d]: expected %s, got %s", id, i, loader.typeString(paramType), loader.typeString(sliceType))
			}
		default:
			if isTimeDuration(paramType) {
				if _, err := durationLiteral(arg.Literal); err != nil {
					return fmt.Errorf("service %q arg[%d]: %w", id, i, err)
				}
				break
			}
			litType, err := literalType(arg.Literal)
			if err != nil {
				return fmt.Errorf("service %q arg[%d]: %w", id, i, err)
			}
			if !types.AssignableTo(litType, paramType) {
				return fmt.Errorf("service %q arg[%d]: expected %s, got %s", id, i, loader.typeString(paramType), loader.typeString(litType))
			}
		}
	}

	return nil
}

func literalType(node yaml.Node) (types.Type, error) {
	switch node.Tag {
	case "!!str":
		return types.Typ[types.UntypedString], nil
	case "!!int":
		return types.Typ[types.UntypedInt], nil
	case "!!bool":
		return types.Typ[types.UntypedBool], nil
	case "!!float":
		return types.Typ[types.UntypedFloat], nil
	case "!!null":
		return types.Typ[types.UntypedNil], nil
	default:
		return nil, fmt.Errorf("unsupported literal type %q", node.Tag)
	}
}
