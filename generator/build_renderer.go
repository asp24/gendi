package generator

import (
	"bytes"
	"fmt"
)

// buildFunctionRenderer renders a build function for a service.
type buildFunctionRenderer interface {
	buildSignature(rnd *Renderer, svc *serviceDef) (signature, innerVar string)
	render(b *bytes.Buffer, rnd *Renderer, ctx *genContext, svc *serviceDef) error
}

// selectBuildRenderer chooses the appropriate renderer based on service properties.
func selectBuildRenderer(svc *serviceDef) buildFunctionRenderer {
	return &regularBuildRenderer{}
}

// regularBuildRenderer renders a standard build function.
type regularBuildRenderer struct{}

func (r *regularBuildRenderer) buildSignature(rnd *Renderer, svc *serviceDef) (string, string) {
	name := rnd.nameGen.buildName(svc)
	retType := rnd.imports.typeString(svc.typeName)
	signature := fmt.Sprintf("func (c *%s) %s() (%s, error)", rnd.containerName, name, retType)
	return signature, ""
}

func (r *regularBuildRenderer) render(b *bytes.Buffer, rnd *Renderer, ctx *genContext, svc *serviceDef) error {
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
