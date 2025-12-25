package generator

import (
	"fmt"
	"go/types"
	"strings"

	"github.com/asp24/gendi/internal/typeutil"
	"github.com/asp24/gendi/ir"
)

// argumentBuilder builds a code expression for a constructor argument
type argumentBuilder interface {
	build(ctx *genContext, svc *serviceDef, arg *ir.Argument, innerVar string, returnsErr bool, argIndex int, paramType types.Type) (expr string, stmts []string, err error)
}

// serviceRefBuilder handles service reference arguments
type serviceRefBuilder struct{}

func (b *serviceRefBuilder) build(ctx *genContext, svc *serviceDef, arg *ir.Argument, innerVar string, returnsErr bool, argIndex int, paramType types.Type) (string, []string, error) {
	dep := ctx.services[arg.Service.ID]
	if dep == nil {
		return "", nil, fmt.Errorf("unknown service %q", arg.Service.ID)
	}
	call := fmt.Sprintf("c.%s()", dep.privateGetterName)
	depVar := ctx.nameGen.varIdent("dep", dep.id)
	if returnsErr {
		stmts := []string{
			fmt.Sprintf("%s, err := %s", depVar, call),
			serviceArgError(svc.id, argIndex),
		}
		return depVar, stmts, nil
	}
	stmts := []string{fmt.Sprintf("%s, _ := %s", depVar, call)}
	return depVar, stmts, nil
}

// innerBuilder handles @.inner decorator arguments
type innerBuilder struct{}

func (b *innerBuilder) build(ctx *genContext, svc *serviceDef, arg *ir.Argument, innerVar string, returnsErr bool, argIndex int, paramType types.Type) (string, []string, error) {
	if innerVar == "" {
		return "", nil, fmt.Errorf("@.inner used outside decorator")
	}
	return innerVar, nil, nil
}

// paramRefBuilder handles parameter reference arguments
type paramRefBuilder struct{}

func (b *paramRefBuilder) build(ctx *genContext, svc *serviceDef, arg *ir.Argument, innerVar string, returnsErr bool, argIndex int, paramType types.Type) (string, []string, error) {
	method := ctx.paramGetters[arg.Parameter.Name]
	if method == "" {
		return "", nil, fmt.Errorf("unknown parameter %q", arg.Parameter.Name)
	}
	paramVar := ctx.nameGen.varIdent("param", arg.Parameter.Name)
	stmts := []string{
		serviceParamNilCheck(svc.id, argIndex, arg.Parameter.Name),
		fmt.Sprintf("%s, err := c.params.%s(%q)", paramVar, method, arg.Parameter.Name),
		serviceParamError(svc.id, argIndex, arg.Parameter.Name),
	}
	return paramVar, stmts, nil
}

// taggedBuilder handles tagged service collection arguments
type taggedBuilder struct{}

func (b *taggedBuilder) build(ctx *genContext, svc *serviceDef, arg *ir.Argument, innerVar string, returnsErr bool, argIndex int, paramType types.Type) (string, []string, error) {
	values := taggedServices(ctx, arg.Tag.Name)
	items := make([]string, 0, len(values))
	stmts := []string{}
	for _, dep := range values {
		call := fmt.Sprintf("c.%s()", dep.privateGetterName)
		varName := ctx.nameGen.varIdent("tag", dep.id)
		if returnsErr {
			stmts = append(stmts, fmt.Sprintf("%s, err := %s", varName, call))
			stmts = append(stmts, serviceTagError(svc.id, argIndex, arg.Tag.Name))
			items = append(items, varName)
		} else {
			stmts = append(stmts, fmt.Sprintf("%s, _ := %s", varName, call))
			items = append(items, varName)
		}
	}
	sliceExpr := "[]" + ctx.imports.typeString(tagElementType(ctx, arg.Tag.Name)) + "{" + strings.Join(items, ", ") + "}"
	return sliceExpr, stmts, nil
}

// literalBuilder handles literal value arguments
type literalBuilder struct{}

func (b *literalBuilder) build(ctx *genContext, svc *serviceDef, arg *ir.Argument, innerVar string, returnsErr bool, argIndex int, paramType types.Type) (string, []string, error) {
	if typeutil.IsDuration(paramType) {
		nanos, err := durationLiteralValue(arg.Literal)
		if err != nil {
			return "", nil, err
		}
		return fmt.Sprintf("%d", nanos), nil, nil
	}
	lit, err := literalValueExpr(arg.Literal)
	if err != nil {
		return "", nil, err
	}
	return lit, nil, nil
}

// getArgumentBuilder returns the appropriate builder for the argument kind
func getArgumentBuilder(kind ir.ArgumentKind) argumentBuilder {
	switch kind {
	case ir.ServiceRefArg:
		return &serviceRefBuilder{}
	case ir.InnerArg:
		return &innerBuilder{}
	case ir.ParamRefArg:
		return &paramRefBuilder{}
	case ir.TaggedArg:
		return &taggedBuilder{}
	default:
		return &literalBuilder{}
	}
}
