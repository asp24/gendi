package generator

import (
	"fmt"
	"go/types"
	"strings"

	"github.com/asp24/gendi/ir"
	"github.com/asp24/gendi/typeres"
)

// argBuildContext bundles parameters for building constructor arguments.
type argBuildContext struct {
	rnd        *ContainerRenderer
	genCtx     *GenContext
	service    *serviceDef
	argument   *ir.Argument
	returnsErr bool
	argIndex   int
	paramType  types.Type
}

// argumentBuilder builds a code expression for a constructor argument
type argumentBuilder interface {
	build(ctx *argBuildContext) (expr string, stmts []string, err error)
}

// serviceRefBuilder handles service reference arguments
type serviceRefBuilder struct{}

func (b *serviceRefBuilder) build(ctx *argBuildContext) (string, []string, error) {
	dep := ctx.genCtx.services[ctx.argument.Service.ID]
	if dep == nil {
		return "", nil, fmt.Errorf("unknown service %q", ctx.argument.Service.ID)
	}

	// Check if slice type conversion is needed
	// This happens when a desugared tag service returns []T but the parameter expects []R
	// where T is assignable to R (e.g., *A to interface{})
	if needsSliceConversion(dep.typeName, ctx.paramType) {
		return b.buildWithSliceConversion(ctx, dep)
	}

	call := fmt.Sprintf("c.%s()", dep.privateGetterName)
	depVar := ctx.rnd.identGenerator.Var(fmt.Sprintf("arg%d", ctx.argIndex), dep.id)
	if ctx.returnsErr {
		stmts := []string{
			fmt.Sprintf("%s, err := %s", depVar, call),
			serviceArgError(ctx.service.id, ctx.argIndex),
		}
		return depVar, stmts, nil
	}
	stmts := []string{fmt.Sprintf("%s, _ := %s", depVar, call)}
	return depVar, stmts, nil
}

// needsSliceConversion checks if slice element type conversion is needed
func needsSliceConversion(svcType, paramType types.Type) bool {
	svcSlice, svcOk := svcType.Underlying().(*types.Slice)
	paramSlice, paramOk := paramType.Underlying().(*types.Slice)
	if !svcOk || !paramOk {
		return false
	}
	// Different element types but svc element assignable to param element
	svcElem := svcSlice.Elem()
	paramElem := paramSlice.Elem()
	return !types.Identical(svcElem, paramElem) && types.AssignableTo(svcElem, paramElem)
}

// buildWithSliceConversion generates code for slice type conversion
func (b *serviceRefBuilder) buildWithSliceConversion(ctx *argBuildContext, dep *serviceDef) (string, []string, error) {
	paramSlice := ctx.paramType.Underlying().(*types.Slice)
	paramElemType := ctx.rnd.importManager.typeString(paramSlice.Elem())
	destType := "[]" + paramElemType

	// For desugared tag services, use tag name for variable naming
	varSuffix := dep.id
	if strings.HasPrefix(dep.id, ir.TagServicePrefix) {
		varSuffix = strings.TrimPrefix(dep.id, ir.TagServicePrefix)
	}

	srcVar := ctx.rnd.identGenerator.Var("tagged", varSuffix)
	destVar := ctx.rnd.identGenerator.Var(fmt.Sprintf("arg%d_tagged", ctx.argIndex), varSuffix)
	call := fmt.Sprintf("c.%s()", dep.privateGetterName)

	var stmts []string
	stmts = append(stmts, fmt.Sprintf("var %s %s", destVar, destType))
	stmts = append(stmts, "{")

	if ctx.returnsErr {
		stmts = append(stmts, fmt.Sprintf("\t%s, err := %s", srcVar, call))
		stmts = append(stmts, serviceArgErrorIndented(ctx.service.id, ctx.argIndex))
	} else {
		stmts = append(stmts, fmt.Sprintf("\t%s, _ := %s", srcVar, call))
	}

	stmts = append(stmts, fmt.Sprintf("\t%s = make(%s, len(%s))", destVar, destType, srcVar))
	stmts = append(stmts, fmt.Sprintf("\tfor idx, item := range %s {", srcVar))
	stmts = append(stmts, fmt.Sprintf("\t\t%s[idx] = item", destVar))
	stmts = append(stmts, "\t}")
	stmts = append(stmts, "}")

	return destVar, stmts, nil
}

// paramRefBuilder handles parameter reference arguments
type paramRefBuilder struct{}

func (b *paramRefBuilder) build(ctx *argBuildContext) (string, []string, error) {
	method := ctx.genCtx.paramGetters[ctx.argument.Parameter.Name]
	if method == "" {
		return "", nil, fmt.Errorf("unknown parameter %q", ctx.argument.Parameter.Name)
	}
	paramVar := ctx.rnd.identGenerator.Var(fmt.Sprintf("param%d", ctx.argIndex), ctx.argument.Parameter.Name)
	stmts := []string{
		// Note: No need to check c.params == nil because the constructor ensures params is never nil
		fmt.Sprintf("%s, err := c.params.%s(%q)", paramVar, method, ctx.argument.Parameter.Name),
		serviceParamError(ctx.service.id, ctx.argIndex, ctx.argument.Parameter.Name),
	}

	// Check if type conversion is needed (named type with basic underlying type)
	paramType := ctx.argument.Parameter.Type
	if named, ok := paramType.(*types.Named); ok {
		// Named type - need to convert from underlying type
		typeStr := ctx.rnd.importManager.typeString(named)
		return fmt.Sprintf("%s(%s)", typeStr, paramVar), stmts, nil
	}

	return paramVar, stmts, nil
}

// literalBuilder handles literal value arguments
type literalBuilder struct{}

func (b *literalBuilder) build(ctx *argBuildContext) (string, []string, error) {
	if typeres.IsDuration(ctx.paramType) {
		nanos, err := durationLiteralValue(ctx.argument.Literal)
		if err != nil {
			return "", nil, err
		}
		return fmt.Sprintf("%d", nanos), nil, nil
	}
	lit, err := literalValueExpr(ctx.argument.Literal)
	if err != nil {
		return "", nil, err
	}
	return lit, nil, nil
}

// spreadBuilder handles spread arguments
type spreadBuilder struct{}

func (b *spreadBuilder) build(ctx *argBuildContext) (string, []string, error) {
	if ctx.argument.Inner == nil {
		return "", nil, fmt.Errorf("spread argument has no inner argument")
	}

	// Build the inner argument expression
	innerCtx := *ctx
	innerCtx.argument = ctx.argument.Inner
	// Use the inner argument's type (which should be a slice)
	innerCtx.paramType = ctx.argument.Inner.Type

	innerBuilder := getArgumentBuilder(ctx.argument.Inner.Kind)
	innerExpr, stmts, err := innerBuilder.build(&innerCtx)
	if err != nil {
		return "", nil, err
	}

	// Add ... to spread the slice
	return innerExpr + "...", stmts, nil
}

// goRefBuilder handles Go package-level variable/constant reference arguments
type goRefBuilder struct{}

func (b *goRefBuilder) build(ctx *argBuildContext) (string, []string, error) {
	obj := ctx.argument.GoRef.Object
	alias := ctx.rnd.importManager.qualifier(obj.Pkg())
	if alias == "" {
		return obj.Name(), nil, nil
	}

	return alias + "." + obj.Name(), nil, nil
}

// fieldAccessBuilder handles field access arguments on services or Go symbols
type fieldAccessBuilder struct{}

func (b *fieldAccessBuilder) build(ctx *argBuildContext) (string, []string, error) {
	fa := ctx.argument.FieldAccess
	fieldChain := strings.Join(fa.FieldNames, ".")

	if fa.Service != nil {
		// Service target: fetch service, then access field chain
		dep := ctx.genCtx.services[fa.Service.ID]
		if dep == nil {
			return "", nil, fmt.Errorf("unknown service %q", fa.Service.ID)
		}

		call := fmt.Sprintf("c.%s()", dep.privateGetterName)
		depVar := ctx.rnd.identGenerator.Var(fmt.Sprintf("arg%d", ctx.argIndex), dep.id)
		if ctx.returnsErr {
			stmts := []string{
				fmt.Sprintf("%s, err := %s", depVar, call),
				serviceArgError(ctx.service.id, ctx.argIndex),
			}
			return depVar + "." + fieldChain, stmts, nil
		}
		stmts := []string{fmt.Sprintf("%s, _ := %s", depVar, call)}
		return depVar + "." + fieldChain, stmts, nil
	}

	if fa.GoRef != nil {
		// Go symbol target: static expression
		obj := fa.GoRef.Object
		alias := ctx.rnd.importManager.qualifier(obj.Pkg())
		var base string
		if alias == "" {
			base = obj.Name()
		} else {
			base = alias + "." + obj.Name()
		}
		return base + "." + fieldChain, nil, nil
	}

	return "", nil, fmt.Errorf("field access has neither service nor go ref target")
}

// argumentBuilderRegistry maps argument kinds to their builder implementations.
// This registry pattern allows adding new argument types without modifying lookup logic.
// Note: TaggedArg is no longer needed as tags are desugared to services in the IR phase.
var argumentBuilderRegistry = map[ir.ArgumentKind]argumentBuilder{
	ir.ServiceRefArg:  &serviceRefBuilder{},
	ir.ParamRefArg:    &paramRefBuilder{},
	ir.LiteralArg:     &literalBuilder{},
	ir.SpreadArg:      &spreadBuilder{},
	ir.GoRefArg:       &goRefBuilder{},
	ir.FieldAccessArg: &fieldAccessBuilder{},
}

// getArgumentBuilder returns the appropriate builder for the argument kind.
// Returns a literal builder as default for unknown argument kinds.
func getArgumentBuilder(kind ir.ArgumentKind) argumentBuilder {
	if builder, ok := argumentBuilderRegistry[kind]; ok {
		return builder
	}
	// Default to literal builder for unknown kinds
	return &literalBuilder{}
}
