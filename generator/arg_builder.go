package generator

import (
	"fmt"
	"go/types"

	"github.com/asp24/gendi/ir"
	"github.com/asp24/gendi/typeres"
)

// argBuildContext bundles parameters for building constructor arguments.
type argBuildContext struct {
	rnd        *ContainerRenderer
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

	// Check if we need slice conversion
	svcType := ctx.argument.Service.Type
	paramType := ctx.paramType

	needsConversion := false
	if !types.AssignableTo(svcType, paramType) {
		// Check for slice compatibility: []T -> []I
		svcSlice, svcIsSlice := svcType.Underlying().(*types.Slice)
		paramSlice, paramIsSlice := paramType.Underlying().(*types.Slice)
		
		if svcIsSlice && paramIsSlice {
			if types.AssignableTo(svcSlice.Elem(), paramSlice.Elem()) || types.Implements(svcSlice.Elem(), paramSlice.Elem().Underlying().(*types.Interface)) {
				needsConversion = true
			}
		}
	}

	call := fmt.Sprintf("c.%s()", dep.privateGetterName)
	
	if needsConversion {
		// Generate conversion loop
		// var argX []ParamElem
		// {
		//    src, _ := c.getSvc()
		//    argX = make([]ParamElem, len(src))
		//    for i, v := range src {
		//        argX[i] = v
		//    }
		// }
		
		paramSlice := paramType.Underlying().(*types.Slice)
		paramElemTypeStr := ctx.rnd.importManager.typeString(paramSlice.Elem())
		destVar := ctx.rnd.identGenerator.Var(fmt.Sprintf("arg%d", ctx.argIndex), dep.id)
		
		stmts := []string{}
		
		// We need a temp scope or just unique vars
		srcVar := ctx.rnd.identGenerator.Var("src", dep.id)
		
		if ctx.returnsErr {
			stmts = append(stmts, fmt.Sprintf("%s, err := %s", srcVar, call))
			stmts = append(stmts, serviceArgError(ctx.service.id, ctx.argIndex))
		} else {
			stmts = append(stmts, fmt.Sprintf("%s, _ := %s", srcVar, call))
		}
		
		stmts = append(stmts, fmt.Sprintf("var %s []%s", destVar, paramElemTypeStr))
		stmts = append(stmts, fmt.Sprintf("%s = make([]%s, len(%s))", destVar, paramElemTypeStr, srcVar))
		stmts = append(stmts, fmt.Sprintf("for i, v := range %s {", srcVar))
		stmts = append(stmts, fmt.Sprintf("\t%s[i] = v", destVar))
		stmts = append(stmts, "}")
		
		return destVar, stmts, nil
	}

	// Standard assignment
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

// argumentBuilderRegistry maps argument kinds to their builder implementations.
// This registry pattern allows adding new argument types without modifying lookup logic.
var argumentBuilderRegistry = map[ir.ArgumentKind]argumentBuilder{
	ir.ServiceRefArg: &serviceRefBuilder{},
	ir.ParamRefArg:   &paramRefBuilder{},
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
