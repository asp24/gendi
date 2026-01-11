package generator

import (
	"bytes"
	"fmt"
	"go/types"
	"strings"

	di "github.com/asp24/gendi"
	"github.com/asp24/gendi/ir"
)

// ContainerRenderer contains tools for rendering generated code.
type ContainerRenderer struct {
	importManager  *ImportManager
	identGenerator *IdentGenerator
	getterRegistry *GetterRegistry
	inliner        Inliner
	containerName  string
}

func NewContainerRenderer(
	imports *ImportManager,
	ident *IdentGenerator,
	getters *GetterRegistry,
	inliner Inliner,
	containerName string,
) *ContainerRenderer {
	return &ContainerRenderer{
		importManager:  imports,
		identGenerator: ident,
		getterRegistry: getters,
		inliner:        inliner,
		containerName:  containerName,
	}
}

func (r *ContainerRenderer) assignNames(ctx *genContext) error {
	if err := r.getterRegistry.Assign(ctx.orderedServiceIDs, ctx.services); err != nil {
		return err
	}

	for id := range ctx.services {
		ctx.services[id].privateGetterName = r.getterRegistry.PrivateService(id)
	}

	return nil
}

func (r *ContainerRenderer) renderContainerStruct(b *bytes.Buffer, ctx *genContext, hasParams bool) error {
	fmt.Fprintf(b, "type %s struct {\n", r.containerName)
	fmt.Fprintf(b, "\tmu sync.Mutex\n")
	fmt.Fprintf(b, "\tparams parameters.Provider\n")
	fmt.Fprintf(b, "\tonMustCallFailed func(serviceName string, err error)\n")

	for _, id := range ctx.orderedServiceIDs {
		svc := ctx.services[id]
		if svc.aliasTarget != "" || !svc.shared {
			continue
		}
		resType := svc.GetterType()
		fmt.Fprintf(b, "\t%s %s\n", r.identGenerator.Field(svc.id), r.importManager.typeString(resType))
		if !isNilable(resType) {
			fmt.Fprintf(b, "\t%sInit bool\n", r.identGenerator.Field(svc.id))
		}
	}
	b.WriteString("}\n\n")

	// ContainerOption type
	fmt.Fprintf(b, "type %sOption func(*%s)\n\n", r.containerName, r.containerName)

	// With<Container>ErrorHandler function
	fmt.Fprintf(b, "func %s(handler func(serviceName string, err error)) %sOption {\n", withErrorHandlerName(r.containerName), r.containerName)
	fmt.Fprintf(b, "\treturn func(c *%s) {\n", r.containerName)
	fmt.Fprintf(b, "\t\tc.onMustCallFailed = handler\n")
	fmt.Fprintf(b, "\t}\n")
	fmt.Fprintf(b, "}\n\n")

	// NewContainer constructor with options
	fmt.Fprintf(b, "func New%s(params parameters.Provider, opts ...%sOption) *%s {\n", r.containerName, r.containerName, r.containerName)
	fmt.Fprintf(b, "\tif params == nil {\n")

	if hasParams {
		fmt.Fprintf(b, "\t\tparams = %s\n", defaultParametersName(r.containerName))
	} else {
		fmt.Fprintf(b, "\t\tparams = parameters.ProviderNullInstance\n")
	}

	fmt.Fprintf(b, "\t}\n")
	fmt.Fprintf(b, "\tc := &%s{\n", r.containerName)
	fmt.Fprintf(b, "\t\tparams: params,\n")
	fmt.Fprintf(b, "\t\tonMustCallFailed: func(string, error) {},\n")
	fmt.Fprintf(b, "\t}\n")
	fmt.Fprintf(b, "\tfor _, opt := range opts {\n")
	fmt.Fprintf(b, "\t\topt(c)\n")
	fmt.Fprintf(b, "\t}\n")
	fmt.Fprintf(b, "\treturn c\n")
	b.WriteString("}\n\n")

	return nil
}

func (r *ContainerRenderer) renderBuildFunctions(b *bytes.Buffer, ctx *genContext) error {
	// Render build functions for each service
	for _, id := range ctx.orderedServiceIDs {
		svc := ctx.services[id]
		if svc.aliasTarget != "" {
			continue
		}
		if err := r.renderBuild(b, ctx, svc); err != nil {
			return err
		}
	}
	return nil
}

func (r *ContainerRenderer) renderGetterFunctions(b *bytes.Buffer, ctx *genContext) error {
	// Render private getters (includes desugared tag services)
	for _, id := range ctx.orderedServiceIDs {
		svc := ctx.services[id]
		if err := r.renderPrivateGetter(b, ctx, svc); err != nil {
			return err
		}
	}

	// Render public getters and Must* methods
	for _, id := range ctx.orderedServiceIDs {
		svc := ctx.services[id]
		if !svc.public {
			continue
		}
		if err := r.renderGetter(b, ctx, svc); err != nil {
			return err
		}
		if err := r.renderMustGetter(b, ctx, svc); err != nil {
			return err
		}
	}

	return nil
}

func (r *ContainerRenderer) renderBuild(b *bytes.Buffer, ctx *genContext, svc *serviceDef) error {
	renderer := selectBuildRenderer(svc)
	return renderer.render(b, r, ctx, svc)
}

func (r *ContainerRenderer) renderPrivateGetter(b *bytes.Buffer, ctx *genContext, svc *serviceDef) error {
	resType := svc.GetterType()
	renderer := selectPrivateGetterRenderer(svc, resType)
	return renderer.render(b, r, ctx, svc)
}

func (r *ContainerRenderer) renderGetter(b *bytes.Buffer, ctx *genContext, svc *serviceDef) error {
	getter := r.getterRegistry.PublicService(svc.id)
	typeStr := r.importManager.typeString(svc.GetterType())

	fmt.Fprintf(b, "func (c *%s) %s() (%s, error) {\n", r.containerName, getter, typeStr)
	fmt.Fprintf(b, "\tc.mu.Lock()\n")
	fmt.Fprintf(b, "\tdefer c.mu.Unlock()\n")
	fmt.Fprintf(b, "\treturn c.%s()\n", svc.privateGetterName)
	b.WriteString("}\n\n")
	return nil
}

func (r *ContainerRenderer) renderMustGetter(b *bytes.Buffer, ctx *genContext, svc *serviceDef) error {
	getter := r.getterRegistry.PublicService(svc.id)
	mustGetter := r.getterRegistry.MustService(svc.id)
	typeStr := r.importManager.typeString(svc.GetterType())

	fmt.Fprintf(b, "func (c *%s) %s() %s {\n", r.containerName, mustGetter, typeStr)
	fmt.Fprintf(b, "\tres, err := c.%s()\n", getter)
	fmt.Fprintf(b, "\tif err != nil {\n")
	fmt.Fprintf(b, "\t\tc.onMustCallFailed(%q, err)\n", svc.id)
	fmt.Fprintf(b, "\t\tpanic(err)\n")
	fmt.Fprintf(b, "\t}\n")
	fmt.Fprintf(b, "\treturn res\n")
	b.WriteString("}\n\n")
	return nil
}

func (r *ContainerRenderer) constructorCall(ctx *genContext, svc *serviceDef, returnsErr bool) ([]string, string, error) {
	var stmts []string
	var args []string
	cons := svc.constructor
	for i, arg := range cons.argDefs {
		var paramType types.Type = types.Typ[types.Invalid]
		if i < len(cons.params) {
			paramType = cons.params[i]
		}
		argExpr, argStmts, err := r.buildArg(ctx, svc, arg, returnsErr, i, paramType)
		if err != nil {
			return nil, "", err
		}
		stmts = append(stmts, argStmts...)
		args = append(args, argExpr)
	}

	if cons.kind == "func" {
		ok, call := r.inliner.TryInline(cons, args)
		if ok {
			return stmts, call, nil
		}

		funcName := r.importManager.funcNameWithTypeArgs(cons.funcObj, cons.typeArgs)

		return stmts, fmt.Sprintf("%s(%s)", funcName, strings.Join(args, ", ")), nil
	}

	recv := cons.methodRecvID
	recvGetter := ctx.services[recv].privateGetterName
	recvExpr := fmt.Sprintf("c.%s()", recvGetter)
	recvVar := r.identGenerator.Var("recv", svc.id)
	if returnsErr {
		stmts = append(stmts, fmt.Sprintf("%s, err := %s", recvVar, recvExpr))
		stmts = append(stmts, serviceReceiverError(svc.id, recv))
	} else {
		stmts = append(stmts, fmt.Sprintf("%s, _ := %s", recvVar, recvExpr))
	}
	recvExpr = recvVar

	return stmts, fmt.Sprintf("%s.%s(%s)", recvExpr, cons.funcObj.Name(), strings.Join(args, ", ")), nil
}

func (r *ContainerRenderer) buildArg(ctx *genContext, svc *serviceDef, arg *ir.Argument, returnsErr bool, argIndex int, paramType types.Type) (string, []string, error) {
	builder := getArgumentBuilder(arg.Kind)
	buildCtx := &argBuildContext{
		rnd:        r,
		genCtx:     ctx,
		service:    svc,
		argument:   arg,
		returnsErr: returnsErr,
		argIndex:   argIndex,
		paramType:  paramType,
	}
	return builder.build(buildCtx)
}

func (r *ContainerRenderer) getterBuildExpr(svc *serviceDef) string {
	return "c." + r.identGenerator.Build(svc.id) + "()"
}

func (r *ContainerRenderer) Render(cfg *di.Config, ctx *genContext, body *bytes.Buffer) error {
	r.importManager.ReserveAliases("sync", "fmt")

	if err := r.assignNames(ctx); err != nil {
		return fmt.Errorf("assign names: %w", err)
	}

	hasParams := len(cfg.Parameters) > 0
	if err := r.renderContainerStruct(body, ctx, hasParams); err != nil {
		return err
	}

	if err := r.renderBuildFunctions(body, ctx); err != nil {
		return err
	}

	if err := r.renderGetterFunctions(body, ctx); err != nil {
		return err
	}

	return nil
}
