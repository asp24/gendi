package generator

import (
	"fmt"
	"go/types"

	"github.com/asp24/gendi/internal/typeutil"
	"github.com/asp24/gendi/ir"
)

// argBuildContext bundles parameters for building constructor arguments.
type argBuildContext struct {
	genCtx     *genContext
	service    *serviceDef
	argument   *ir.Argument
	innerVar   string
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
	call := fmt.Sprintf("c.%s()", dep.privateGetterName)
	depVar := ctx.genCtx.nameGen.varIdent("dep", dep.id)
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

// innerBuilder handles @.inner decorator arguments
type innerBuilder struct{}

func (b *innerBuilder) build(ctx *argBuildContext) (string, []string, error) {
	if ctx.innerVar == "" {
		return "", nil, fmt.Errorf("@.inner used outside decorator")
	}
	return ctx.innerVar, nil, nil
}

// paramRefBuilder handles parameter reference arguments
type paramRefBuilder struct{}

func (b *paramRefBuilder) build(ctx *argBuildContext) (string, []string, error) {
	method := ctx.genCtx.paramGetters[ctx.argument.Parameter.Name]
	if method == "" {
		return "", nil, fmt.Errorf("unknown parameter %q", ctx.argument.Parameter.Name)
	}
	paramVar := ctx.genCtx.nameGen.varIdent("param", ctx.argument.Parameter.Name)
	stmts := []string{
		// Note: No need to check c.params == nil because the constructor ensures params is never nil
		fmt.Sprintf("%s, err := c.params.%s(%q)", paramVar, method, ctx.argument.Parameter.Name),
		serviceParamError(ctx.service.id, ctx.argIndex, ctx.argument.Parameter.Name),
	}

	// Check if type conversion is needed (named type with basic underlying type)
	paramType := ctx.argument.Parameter.Type
	if named, ok := paramType.(*types.Named); ok {
		// Named type - need to convert from underlying type
		typeStr := ctx.genCtx.imports.typeString(named)
		return fmt.Sprintf("%s(%s)", typeStr, paramVar), stmts, nil
	}

	return paramVar, stmts, nil
}

// taggedBuilder handles tagged service collection arguments
type taggedBuilder struct{}

func (b *taggedBuilder) build(ctx *argBuildContext) (string, []string, error) {
	tagName := ctx.argument.Tag.Name
	getter := ctx.genCtx.nameGen.privateTagGetterName(tagName)
	if getter == "" {
		return "", nil, fmt.Errorf("tag %q: missing private getter", tagName)
	}

	varName := ctx.genCtx.nameGen.varIdent(fmt.Sprintf("tagged%d", ctx.argIndex), tagName)
	call := fmt.Sprintf("c.%s()", getter)
	stmts := []string{}
	if ctx.returnsErr {
		stmts = append(stmts, fmt.Sprintf("%s, err := %s", varName, call))
		stmts = append(stmts, serviceTagError(ctx.service.id, ctx.argIndex, tagName))
	} else {
		stmts = append(stmts, fmt.Sprintf("%s, _ := %s", varName, call))
	}

	elemType := tagElementType(ctx.genCtx, tagName)
	paramSlice, ok := ctx.paramType.Underlying().(*types.Slice)
	if !ok {
		return "", nil, fmt.Errorf("service %q arg[%d]: tagged injection requires slice type, got %s", ctx.service.id, ctx.argIndex, ctx.paramType)
	}
	paramElem := paramSlice.Elem()

	if types.Identical(elemType, paramElem) {
		return varName, stmts, nil
	}

	convVar := ctx.genCtx.nameGen.varIdent(fmt.Sprintf("taggedConv%d", ctx.argIndex), tagName)
	convType := "[]" + ctx.genCtx.imports.typeString(paramElem)
	idxVar := ctx.genCtx.nameGen.varIdent(fmt.Sprintf("tagIdx%d", ctx.argIndex), tagName)
	itemVar := ctx.genCtx.nameGen.varIdent(fmt.Sprintf("tagItem%d", ctx.argIndex), tagName)
	stmts = append(stmts, fmt.Sprintf("%s := make(%s, len(%s))", convVar, convType, varName))
	stmts = append(stmts, fmt.Sprintf("for %s, %s := range %s {", idxVar, itemVar, varName))
	stmts = append(stmts, fmt.Sprintf("\t%s[%s] = %s", convVar, idxVar, itemVar))
	stmts = append(stmts, "}")
	return convVar, stmts, nil
}

// literalBuilder handles literal value arguments
type literalBuilder struct{}

func (b *literalBuilder) build(ctx *argBuildContext) (string, []string, error) {
	if typeutil.IsDuration(ctx.paramType) {
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

// argumentBuilderRegistry maps argument kinds to their builder implementations.
// This registry pattern allows adding new argument types without modifying lookup logic.
var argumentBuilderRegistry = map[ir.ArgumentKind]argumentBuilder{
	ir.ServiceRefArg: &serviceRefBuilder{},
	ir.InnerArg:      &innerBuilder{},
	ir.ParamRefArg:   &paramRefBuilder{},
	ir.TaggedArg:     &taggedBuilder{},
	ir.LiteralArg:    &literalBuilder{},
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
