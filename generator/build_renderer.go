package generator

import (
	"bytes"
	"fmt"

	"github.com/asp24/gendi/ir"
)

// buildFunctionRenderer renders a build function for a service.
type buildFunctionRenderer interface {
	buildSignature(rnd *ContainerRenderer, svc *serviceDef) (signature, innerVar string)
	render(b *bytes.Buffer, rnd *ContainerRenderer, ctx *genContext, svc *serviceDef) error
}

// selectBuildRenderer chooses the appropriate renderer based on service properties.
func selectBuildRenderer(svc *serviceDef) buildFunctionRenderer {
	return &regularBuildRenderer{}
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

// regularBuildRenderer renders a standard build function.
type regularBuildRenderer struct{}

func (r *regularBuildRenderer) buildSignature(rnd *ContainerRenderer, svc *serviceDef) (string, string) {
	name := rnd.ident.Build(svc.id)
	retType := rnd.imports.typeString(svc.typeName)
	signature := fmt.Sprintf("func (c *%s) %s() (%s, error)", rnd.containerName, name, retType)
	return signature, ""
}

func (r *regularBuildRenderer) render(b *bytes.Buffer, rnd *ContainerRenderer, ctx *genContext, svc *serviceDef) error {
	signature, innerVar := r.buildSignature(rnd, svc)
	retType := rnd.imports.typeString(svc.typeName)
	returnsErr := buildNeedsErrorHandling(svc)

	fmt.Fprintf(b, "%s {\n", signature)
	if returnsErr {
		fmt.Fprintf(b, "\tvar zero %s\n", retType)
	}

	stmts, callExpr, err := rnd.constructorCall(ctx, svc, innerVar, returnsErr)
	if err != nil {
		return err
	}
	for _, stmt := range stmts {
		fmt.Fprintf(b, "\t%s\n", stmt)
	}
	if svc.constructor.returnsError {
		fmt.Fprintf(b, "\tres, err := %s\n", callExpr)
		fmt.Fprintf(b, "\t%s\n", serviceConstructorError(svc.id))
		fmt.Fprintf(b, "\treturn res, nil\n")
	} else {
		fmt.Fprintf(b, "\treturn %s, nil\n", callExpr)
	}
	b.WriteString("}\n\n")
	return nil
}
