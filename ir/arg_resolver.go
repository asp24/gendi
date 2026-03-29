package ir

import (
	"fmt"
	"go/types"
	"strings"

	di "github.com/asp24/gendi"
	"github.com/asp24/gendi/srcloc"
	"github.com/asp24/gendi/typeres"
	"github.com/asp24/gendi/yaml"
)

// argResolver resolves constructor arguments
type argResolver struct {
	typeResolver TypeResolver
}

// resolve resolves a single constructor argument
func (r *argResolver) resolve(container *Container, svcID string, idx int, arg di.Argument, paramType types.Type) (*Argument, error) {
	switch arg.Kind {
	case di.ArgServiceRef:
		return r.resolveServiceRef(container, svcID, idx, arg, paramType)
	case di.ArgInner:
		return nil, srcloc.Errorf(arg.SourceLoc, "service %q arg[%d]: @.inner should have been resolved by DecoratorPass", svcID, idx)
	case di.ArgParam:
		return r.resolveParam(container, svcID, idx, arg, paramType)
	case di.ArgTagged:
		return r.resolveTagged(container, svcID, idx, arg, paramType)
	case di.ArgSpread:
		return r.resolveSpread(container, svcID, idx, arg, paramType)
	case di.ArgFieldAccessService:
		return r.resolveFieldAccessServiceArg(container, svcID, idx, arg, paramType)
	case di.ArgFieldAccessGo:
		return r.resolveFieldAccessGoArg(svcID, idx, arg, paramType)
	case di.ArgGoRef:
		return r.resolveGoRef(svcID, idx, arg, paramType)
	case di.ArgLiteral:
		return r.resolveLiteral(svcID, idx, arg, paramType)
	default:
		return nil, srcloc.Errorf(arg.SourceLoc, "service %q arg[%d]: unknown argument kind %d", svcID, idx, arg.Kind)
	}
}

func (r *argResolver) resolveServiceRef(container *Container, svcID string, idx int, arg di.Argument, paramType types.Type) (*Argument, error) {
	dep, ok := container.Services[arg.Value]
	if !ok {
		return nil, srcloc.Errorf(arg.SourceLoc, "service %q arg[%d]: unknown service %q", svcID, idx, arg.Value)
	}
	return &Argument{Type: paramType, Kind: ServiceRefArg, Service: dep}, nil
}

func (r *argResolver) resolveParam(container *Container, svcID string, idx int, arg di.Argument, paramType types.Type) (*Argument, error) {
	param, ok := container.Parameters[arg.Value]
	if !ok {
		// Parameter might be provided at runtime
		return &Argument{
			Type:      paramType,
			Kind:      ParamRefArg,
			Parameter: &Parameter{Name: arg.Value, Type: paramType},
		}, nil
	}
	if !types.AssignableTo(param.Type, paramType) {
		return nil, srcloc.Errorf(arg.SourceLoc, "service %q arg[%d]: parameter %q type mismatch", svcID, idx, arg.Value)
	}
	return &Argument{Type: paramType, Kind: ParamRefArg, Parameter: param}, nil
}

func (r *argResolver) resolveTagged(container *Container, svcID string, idx int, arg di.Argument, paramType types.Type) (*Argument, error) {
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
		return nil, srcloc.Errorf(arg.SourceLoc, "service %q arg[%d]: tagged injection requires slice type, got %s", svcID, idx, paramType)
	}
	elemType := sliceType.Elem()

	if tag.ElementType == nil {
		// Infer ElementType from parameter
		tag.ElementType = elemType
	} else if !types.AssignableTo(tag.ElementType, elemType) {
		// Validate consistency if ElementType was declared or inferred earlier
		return nil, srcloc.Errorf(arg.SourceLoc, "service %q arg[%d]: tag %q element type mismatch: %s is not assignable to %s",
			svcID, idx, arg.Value, tag.ElementType, elemType)
	}

	return &Argument{Type: paramType, Kind: TaggedArg, Tag: tag}, nil
}

func (r *argResolver) resolveSpread(container *Container, svcID string, idx int, arg di.Argument, paramType types.Type) (*Argument, error) {
	// Validate that parameter is variadic (represented as slice in go/types)
	if _, ok := paramType.Underlying().(*types.Slice); !ok {
		return nil, srcloc.Errorf(arg.SourceLoc, "service %q arg[%d]: !spread: can only be used with variadic parameters, got %s", svcID, idx, paramType)
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
		return nil, srcloc.WrapError(arg.SourceLoc, fmt.Sprintf("service %q arg[%d]: !spread", svcID, idx), err)
	}

	// Validate that inner resolved to a slice type
	if _, ok := innerResolved.Type.Underlying().(*types.Slice); !ok {
		return nil, srcloc.Errorf(arg.SourceLoc, "service %q arg[%d]: !spread: requires slice type, got %s", svcID, idx, innerResolved.Type)
	}

	return &Argument{Type: paramType, Kind: SpreadArg, Inner: innerResolved}, nil
}

func (r *argResolver) resolveFieldAccessServiceArg(container *Container, svcID string, idx int, arg di.Argument, paramType types.Type) (*Argument, error) {
	fa, err := r.resolveFieldAccessOnService(container, svcID, idx, arg, arg.Value)
	if err != nil {
		return nil, err
	}
	if !types.AssignableTo(fa.ResultType, paramType) {
		return nil, srcloc.Errorf(arg.SourceLoc, "service %q arg[%d]: field access result type %s is not assignable to %s",
			svcID, idx, fa.ResultType, paramType)
	}
	return &Argument{Type: paramType, Kind: FieldAccessArg, FieldAccess: fa}, nil
}

func (r *argResolver) resolveFieldAccessGoArg(svcID string, idx int, arg di.Argument, paramType types.Type) (*Argument, error) {
	fa, err := r.resolveFieldAccessOnGoRef(svcID, idx, arg, arg.Value)
	if err != nil {
		return nil, err
	}
	if !types.AssignableTo(fa.ResultType, paramType) {
		return nil, srcloc.Errorf(arg.SourceLoc, "service %q arg[%d]: field access result type %s is not assignable to %s",
			svcID, idx, fa.ResultType, paramType)
	}
	return &Argument{Type: paramType, Kind: FieldAccessArg, FieldAccess: fa}, nil
}

func (r *argResolver) resolveGoRef(svcID string, idx int, arg di.Argument, paramType types.Type) (*Argument, error) {
	pkgPath, name, _, err := typeres.SplitQualifiedNameWithTypeParams(arg.Value)
	if err != nil {
		return nil, srcloc.WrapError(arg.SourceLoc, fmt.Sprintf("service %q arg[%d]: invalid go reference %q", svcID, idx, arg.Value), err)
	}
	obj, err := r.typeResolver.LookupVar(pkgPath, name)
	if err != nil {
		return nil, srcloc.WrapError(arg.SourceLoc, fmt.Sprintf("service %q arg[%d]", svcID, idx), err)
	}
	if !types.AssignableTo(obj.Type(), paramType) {
		return nil, srcloc.Errorf(arg.SourceLoc, "service %q arg[%d]: go reference %q type %s is not assignable to %s",
			svcID, idx, arg.Value, obj.Type(), paramType)
	}
	return &Argument{Type: paramType, Kind: GoRefArg, GoRef: &GoRef{Object: obj}}, nil
}

func (r *argResolver) resolveLiteral(svcID string, idx int, arg di.Argument, paramType types.Type) (*Argument, error) {
	litVal, err := convertLiteral(arg.Literal, paramType)
	if err != nil {
		return nil, srcloc.WrapError(arg.SourceLoc, fmt.Sprintf("service %q arg[%d]", svcID, idx), err)
	}
	return &Argument{Type: paramType, Kind: LiteralArg, Literal: litVal}, nil
}

// resolveFieldAccessOnService resolves !field:@service.Field.Chain.
// Uses longest prefix match to find the service ID, since service IDs can contain dots.
func (r *argResolver) resolveFieldAccessOnService(container *Container, svcID string, idx int, arg di.Argument, value string) (*FieldAccess, error) {
	parts := strings.Split(value, ".")
	if len(parts) < 2 {
		return nil, srcloc.Errorf(arg.SourceLoc, "service %q arg[%d]: !field:@%s requires at least one field name", svcID, idx, value)
	}

	// Longest prefix match: try progressively shorter prefixes
	var service *Service
	var fieldParts []string
	for i := len(parts) - 1; i >= 1; i-- {
		candidate := strings.Join(parts[:i], ".")
		if svc, ok := container.Services[candidate]; ok {
			service = svc
			fieldParts = parts[i:]
			break
		}
	}
	if service == nil {
		return nil, srcloc.Errorf(arg.SourceLoc, "service %q arg[%d]: !field:@%s: no matching service found", svcID, idx, value)
	}

	baseType := service.Type
	resultType, err := walkFieldChain(baseType, fieldParts)
	if err != nil {
		return nil, srcloc.WrapError(arg.SourceLoc, fmt.Sprintf("service %q arg[%d]: !field:@%s", svcID, idx, value), err)
	}

	return &FieldAccess{
		Service:    service,
		FieldNames: fieldParts,
		ResultType: resultType,
	}, nil
}

// resolveFieldAccessOnGoRef resolves !field:!go:pkg.Symbol.Field.Chain.
// Iteratively strips trailing dot-separated parts and tries LookupVar until
// the package-level symbol is found. The remainder becomes the field chain.
func (r *argResolver) resolveFieldAccessOnGoRef(svcID string, idx int, arg di.Argument, value string) (*FieldAccess, error) {
	parts := strings.Split(value, ".")
	if len(parts) < 3 {
		// Need at least pkg.Symbol.Field
		return nil, srcloc.Errorf(arg.SourceLoc, "service %q arg[%d]: !field:!go:%s requires at least one field name after the symbol", svcID, idx, value)
	}

	// Try progressively shorter prefixes to find the package-level symbol.
	// Minimum 2 parts for a valid qualified name (pkg.Name).
	var obj types.Object
	var fieldParts []string
	for i := len(parts) - 1; i >= 2; i-- {
		qualName := strings.Join(parts[:i], ".")
		pkgPath, name, _, err := typeres.SplitQualifiedNameWithTypeParams(qualName)
		if err != nil {
			continue
		}
		obj, err = r.typeResolver.LookupVar(pkgPath, name)
		if err != nil {
			continue
		}
		fieldParts = parts[i:]
		break
	}
	if obj == nil {
		return nil, srcloc.Errorf(arg.SourceLoc, "service %q arg[%d]: !field:!go:%s: no matching package-level symbol found", svcID, idx, value)
	}

	baseType := obj.Type()
	resultType, err := walkFieldChain(baseType, fieldParts)
	if err != nil {
		return nil, srcloc.WrapError(arg.SourceLoc, fmt.Sprintf("service %q arg[%d]: !field:!go:%s", svcID, idx, value), err)
	}

	return &FieldAccess{
		GoRef:      &GoRef{Object: obj},
		FieldNames: fieldParts,
		ResultType: resultType,
	}, nil
}

// walkFieldChain walks a chain of field names on a type, returning the final field's type.
// Pointer types are automatically dereferenced at each level.
func walkFieldChain(baseType types.Type, fieldNames []string) (types.Type, error) {
	if len(fieldNames) == 0 {
		return nil, fmt.Errorf("empty field chain")
	}

	currentType := baseType
	for _, fieldName := range fieldNames {
		// Dereference pointer types
		for {
			ptr, ok := currentType.(*types.Pointer)
			if !ok {
				break
			}
			currentType = ptr.Elem()
		}

		obj, _, _ := types.LookupFieldOrMethod(currentType, true, nil, fieldName)
		if obj == nil {
			return nil, fmt.Errorf("field %q not found on type %s", fieldName, currentType)
		}
		field, ok := obj.(*types.Var)
		if !ok || !field.IsField() {
			return nil, fmt.Errorf("%q is not a field on type %s", fieldName, currentType)
		}
		if !field.Exported() {
			return nil, fmt.Errorf("field %q on type %s is not exported", fieldName, currentType)
		}
		currentType = field.Type()
	}
	return currentType, nil
}
