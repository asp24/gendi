package generator

import (
	"bytes"
	"fmt"
	"go/types"
	"sort"
	"strconv"
	"strings"

	di "github.com/asp24/gendi"
	"github.com/asp24/gendi/ir"
)

func collectTagGetterNames(ctx *genContext) []string {
	tagNames := map[string]bool{}
	for name, tag := range ctx.tags {
		if tag != nil && tag.Public {
			tagNames[name] = true
		}
	}
	for _, svc := range ctx.services {
		if svc == nil {
			continue
		}
		for _, arg := range svc.constructor.argDefs {
			if arg.Kind == ir.TaggedArg && arg.Tag != nil {
				tagNames[arg.Tag.Name] = true
			}
		}
	}
	items := make([]string, 0, len(tagNames))
	for name := range tagNames {
		items = append(items, name)
	}
	sort.Strings(items)
	return items
}

func (g *Generator) render(ctx *genContext, rnd *Renderer) ([]byte, error) {
	// Assign getter names and populate serviceDef fields
	tagGetterNames := collectTagGetterNames(ctx)
	rnd.assignNames(ctx, tagGetterNames)

	body := &bytes.Buffer{}
	hasParams := len(g.cfg.Parameters) > 0
	reservedAliases := []string{"sync", "fmt"}
	if hasParams {
		reservedAliases = append(reservedAliases, "parameters")
	}
	rnd.imports.ReserveAliases(reservedAliases...)

	// Render main code sections
	if err := g.renderParameters(body, hasParams); err != nil {
		return nil, err
	}
	if err := rnd.renderContainerStruct(body, ctx, hasParams); err != nil {
		return nil, err
	}
	if err := rnd.renderBuildFunctions(body, ctx); err != nil {
		return nil, err
	}
	if err := rnd.renderGetterFunctions(body, ctx, tagGetterNames); err != nil {
		return nil, err
	}

	// Assemble final output with header
	return g.assembleOutput(body, rnd, hasParams), nil
}

func (r *Renderer) assignNames(ctx *genContext, tagGetterNames []string) {
	r.getters.Assign(ctx.orderedServiceIDs, ctx.services, ctx.tags, tagGetterNames)
	for id := range ctx.services {
		if ctx.services[id].public {
			ctx.services[id].getterName = r.getters.PublicService(id)
		}
		ctx.services[id].privateGetterName = r.getters.PrivateService(id)
	}
}

func (g *Generator) renderParameters(b *bytes.Buffer, hasParams bool) error {
	if !hasParams {
		return nil
	}

	keys := make([]string, 0, len(g.cfg.Parameters))
	for k := range g.cfg.Parameters {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	b.WriteString("var DefaultParameters = parameters.NewProviderMap(map[string]any{\n")
	for _, name := range keys {
		param := g.cfg.Parameters[name]
		lit, err := literalExpr(param.Value)
		if err != nil {
			return fmt.Errorf("parameter %q: %w", name, err)
		}
		fmt.Fprintf(b, "\t%q: %s,\n", name, lit)
	}
	b.WriteString("})\n\n")
	return nil
}

func (r *Renderer) renderContainerStruct(b *bytes.Buffer, ctx *genContext, hasParams bool) error {
	fmt.Fprintf(b, "type %s struct {\n", r.containerName)
	fmt.Fprintf(b, "\tmu sync.Mutex\n")
	if hasParams {
		fmt.Fprintf(b, "\tparams parameters.Provider\n")
	}

	for _, id := range ctx.orderedServiceIDs {
		svc := ctx.services[id]
		if svc.aliasTarget != "" || !svc.shared {
			continue
		}
		resType := getterType(svc)
		nilable := isNilable(resType)
		fmt.Fprintf(b, "\t%s %s\n", r.ident.Field(svc.id), r.imports.typeString(resType))
		if !nilable {
			fmt.Fprintf(b, "\t%sInit bool\n", r.ident.Field(svc.id))
		}
	}
	b.WriteString("}\n\n")

	if hasParams {
		fmt.Fprintf(b, "func New%s(params parameters.Provider) *%s {\n", r.containerName, r.containerName)
		fmt.Fprintf(b, "\tif params == nil {\n")
		fmt.Fprintf(b, "\t\tparams = DefaultParameters\n")
		fmt.Fprintf(b, "\t}\n")
		fmt.Fprintf(b, "\treturn &%s{params: params}\n", r.containerName)
		b.WriteString("}\n\n")
	}
	return nil
}

func (r *Renderer) renderBuildFunctions(b *bytes.Buffer, ctx *genContext) error {
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

func (r *Renderer) renderGetterFunctions(b *bytes.Buffer, ctx *genContext, tagGetterNames []string) error {
	// Render private getters
	for _, id := range ctx.orderedServiceIDs {
		svc := ctx.services[id]
		if err := r.renderPrivateGetter(b, ctx, svc); err != nil {
			return err
		}
	}

	// Render private tag getters
	for _, name := range tagGetterNames {
		tag := ctx.tags[name]
		if tag == nil {
			continue
		}
		if err := r.renderPrivateTagGetter(b, ctx, name, tag); err != nil {
			return err
		}
	}

	// Render public getters
	for _, id := range ctx.orderedServiceIDs {
		svc := ctx.services[id]
		if !svc.public {
			continue
		}
		if err := r.renderGetter(b, ctx, svc); err != nil {
			return err
		}
	}

	// Render public tag getters
	for _, name := range tagGetterNames {
		tag := ctx.tags[name]
		if tag == nil || !tag.Public {
			continue
		}
		if err := r.renderTagGetter(b, ctx, name, tag); err != nil {
			return err
		}
	}
	return nil
}

func (g *Generator) assembleOutput(body *bytes.Buffer, rnd *Renderer, hasParams bool) []byte {
	out := &bytes.Buffer{}

	// Build tags
	if g.options.BuildTags != "" {
		fmt.Fprintf(out, "//go:build %s\n", g.options.BuildTags)
		fmt.Fprintf(out, "// +build %s\n\n", g.options.BuildTags)
	}

	// Package and imports
	fmt.Fprintln(out, "// Code generated by di-gen; DO NOT EDIT.")
	fmt.Fprintf(out, "package %s\n\n", g.options.Package)

	extraImports := []string{"sync", "fmt"}
	if hasParams {
		extraImports = append(extraImports, "github.com/asp24/gendi/parameters")
	}
	out.WriteString(rnd.imports.renderImports(extraImports))
	out.Write(body.Bytes())

	return out.Bytes()
}

func (r *Renderer) renderBuild(b *bytes.Buffer, ctx *genContext, svc *serviceDef) error {
	renderer := selectBuildRenderer(svc)
	return renderer.render(b, r, ctx, svc)
}

func (r *Renderer) renderPrivateGetter(b *bytes.Buffer, ctx *genContext, svc *serviceDef) error {
	resType := getterType(svc)
	renderer := selectPrivateGetterRenderer(svc, resType)
	return renderer.render(b, r, ctx, svc)
}

func (r *Renderer) renderGetter(b *bytes.Buffer, ctx *genContext, svc *serviceDef) error {
	getter := svc.getterName
	fmt.Fprintf(b, "func (c *%s) %s() (%s, error) {\n", r.containerName, getter, r.imports.typeString(getterType(svc)))
	fmt.Fprintf(b, "\tc.mu.Lock()\n")
	fmt.Fprintf(b, "\tdefer c.mu.Unlock()\n")
	fmt.Fprintf(b, "\treturn c.%s()\n", svc.privateGetterName)
	b.WriteString("}\n\n")
	return nil
}

func (r *Renderer) renderPrivateTagGetter(b *bytes.Buffer, ctx *genContext, tagName string, tag *ir.Tag) error {
	if tag.ElementType == nil {
		return fmt.Errorf("tag %q element type is required for tag getter", tagName)
	}
	getter := r.getters.PrivateTag(tagName)
	elemType := r.imports.typeString(tag.ElementType)
	sliceType := "[]" + elemType
	items := taggedServices(ctx, tagName)

	fmt.Fprintf(b, "func (c *%s) %s() (%s, error) {\n", r.containerName, getter, sliceType)
	fmt.Fprintf(b, "\titems := make(%s, 0, %d)\n", sliceType, len(items))
	for _, svc := range items {
		varName := r.ident.Var("tagged", svc.id)
		fmt.Fprintf(b, "\t%s, err := c.%s()\n", varName, svc.privateGetterName)
		fmt.Fprintf(b, "\tif err != nil {\n\t\treturn nil, err\n\t}\n")
		fmt.Fprintf(b, "\titems = append(items, %s)\n", varName)
	}
	fmt.Fprintf(b, "\treturn items, nil\n")
	b.WriteString("}\n\n")
	return nil
}

func (r *Renderer) renderTagGetter(b *bytes.Buffer, ctx *genContext, tagName string, tag *ir.Tag) error {
	getter := r.getters.PublicTag(tagName)
	privateGetter := r.getters.PrivateTag(tagName)
	elemType := r.imports.typeString(tag.ElementType)
	sliceType := "[]" + elemType
	fmt.Fprintf(b, "func (c *%s) %s() (%s, error) {\n", r.containerName, getter, sliceType)
	fmt.Fprintf(b, "\tc.mu.Lock()\n")
	fmt.Fprintf(b, "\tdefer c.mu.Unlock()\n")
	fmt.Fprintf(b, "\treturn c.%s()\n", privateGetter)
	b.WriteString("}\n\n")
	return nil
}

func (r *Renderer) constructorCall(ctx *genContext, svc *serviceDef, innerVar string, returnsErr bool) ([]string, string, error) {
	var stmts []string
	var args []string
	for i, arg := range svc.constructor.argDefs {
		var paramType types.Type = types.Typ[types.Invalid]
		if i < len(svc.constructor.params) {
			paramType = svc.constructor.params[i]
		}
		argExpr, argStmts, err := r.buildArg(ctx, svc, arg, innerVar, returnsErr, i, paramType)
		if err != nil {
			return nil, "", err
		}
		stmts = append(stmts, argStmts...)
		args = append(args, argExpr)
	}

	var call string
	if svc.constructor.kind == "func" {
		funcName := r.imports.funcNameWithTypeArgs(svc.constructor.funcObj, svc.constructor.typeArgs)
		call = fmt.Sprintf("%s(%s)", funcName, strings.Join(args, ", "))
	} else {
		recv := svc.constructor.methodRecvID
		recvGetter := ctx.services[recv].privateGetterName
		recvExpr := fmt.Sprintf("c.%s()", recvGetter)
		recvVar := r.ident.Var("recv", svc.id)
		if returnsErr {
			stmts = append(stmts, fmt.Sprintf("%s, err := %s", recvVar, recvExpr))
			stmts = append(stmts, serviceReceiverError(svc.id, recv))
		} else {
			stmts = append(stmts, fmt.Sprintf("%s, _ := %s", recvVar, recvExpr))
		}
		recvExpr = recvVar
		call = fmt.Sprintf("%s.%s(%s)", recvExpr, svc.constructor.funcObj.Name(), strings.Join(args, ", "))
	}
	return stmts, call, nil
}

func (r *Renderer) buildArg(ctx *genContext, svc *serviceDef, arg *ir.Argument, innerVar string, returnsErr bool, argIndex int, paramType types.Type) (string, []string, error) {
	builder := getArgumentBuilder(arg.Kind)
	buildCtx := &argBuildContext{
		rnd:        r,
		genCtx:     ctx,
		service:    svc,
		argument:   arg,
		innerVar:   innerVar,
		returnsErr: returnsErr,
		argIndex:   argIndex,
		paramType:  paramType,
	}
	return builder.build(buildCtx)
}

func taggedServices(ctx *genContext, tag string) []*serviceDef {
	var items []*serviceDef
	for id, svc := range ctx.services {
		for _, t := range svc.tags {
			if t.Tag.Name == tag {
				items = append(items, ctx.services[id])
			}
		}
	}
	sortByPriority := false
	if def, ok := ctx.tags[tag]; ok && def.SortBy == "priority" {
		sortByPriority = true
	}
	if sortByPriority {
		sort.Slice(items, func(i, j int) bool {
			pi, pj := tagPriority(items[i], tag), tagPriority(items[j], tag)
			if pi == pj {
				return items[i].id < items[j].id
			}
			return pi > pj
		})
	} else {
		sort.Slice(items, func(i, j int) bool {
			return items[i].id < items[j].id
		})
	}
	return items
}

func tagPriority(svc *serviceDef, tag string) int {
	for _, t := range svc.tags {
		if t.Tag.Name != tag {
			continue
		}
		if v, ok := t.Attributes["priority"]; ok {
			switch val := v.(type) {
			case int:
				return val
			case int64:
				return int(val)
			case float64:
				return int(val)
			case string:
				if parsed, err := strconv.Atoi(val); err == nil {
					return parsed
				}
			}
		}
	}
	return 0
}

func tagElementType(ctx *genContext, tag string) types.Type {
	if t, ok := ctx.tags[tag]; ok {
		return t.ElementType
	}
	return types.Typ[types.Invalid]
}

func (r *Renderer) getterBuildExpr(svc *serviceDef) string {
	return "c." + r.ident.Build(svc.id) + "()"
}

func literalExpr(lit di.Literal) (string, error) {
	switch lit.Kind {
	case di.LiteralString:
		return strconv.Quote(lit.String()), nil
	case di.LiteralInt:
		return fmt.Sprintf("%d", lit.Int()), nil
	case di.LiteralFloat:
		return fmt.Sprintf("%v", lit.Float()), nil
	case di.LiteralBool:
		return fmt.Sprintf("%t", lit.Bool()), nil
	case di.LiteralNull:
		return "nil", nil
	default:
		return "", fmt.Errorf("unsupported literal kind %d", lit.Kind)
	}
}

func buildNeedsErrorHandling(svc *serviceDef) bool {
	if svc.constructor.returnsError || svc.constructor.kind == "method" {
		return true
	}
	for _, arg := range svc.constructor.argDefs {
		switch arg.Kind {
		case ir.ServiceRefArg, ir.TaggedArg, ir.ParamRefArg:
			return true
		}
	}
	return false
}
