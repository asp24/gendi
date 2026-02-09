package generator

import (
	"bytes"
	"fmt"
	"go/types"
)

// privateGetterRenderer renders a private getter function for a service.
type privateGetterRenderer interface {
	render(b *bytes.Buffer, rnd *ContainerRenderer, ctx *GenContext, svc *serviceDef) error
}

// selectPrivateGetterRenderer chooses the appropriate renderer based on service properties.
func selectPrivateGetterRenderer(svc *serviceDef, resType types.Type) privateGetterRenderer {
	if svc.IsAlias() {
		return &aliasGetterRenderer{}
	}
	if !svc.shared {
		return &nonSharedGetterRenderer{}
	}
	if isNilable(resType) {
		return &sharedPtrGetterRenderer{}
	}
	return &sharedValueGetterRenderer{}
}

// aliasGetterRenderer renders a getter that delegates to an alias target.
type aliasGetterRenderer struct{}

func (g *aliasGetterRenderer) render(b *bytes.Buffer, rnd *ContainerRenderer, ctx *GenContext, svc *serviceDef) error {
	getter := svc.privateGetterName
	getterTypeStr := rnd.importManager.typeString(svc.GetterType())

	target := ctx.services[svc.aliasTarget]
	if target == nil {
		return fmt.Errorf("unknown alias target %q", svc.aliasTarget)
	}

	fmt.Fprintf(b, "func (c *%s) %s() (%s, error) {\n", rnd.containerName, getter, getterTypeStr)
	fmt.Fprintf(b, "\treturn c.%s()\n", target.privateGetterName)
	b.WriteString("}\n\n")
	return nil
}

// sharedPtrGetterRenderer renders a getter for shared services with pointer types.
// Uses the field directly for caching (nil check).
type sharedPtrGetterRenderer struct{}

func (g *sharedPtrGetterRenderer) render(b *bytes.Buffer, rnd *ContainerRenderer, _ *GenContext, svc *serviceDef) error {
	getter := svc.privateGetterName
	getterTypeStr := rnd.importManager.typeString(svc.GetterType())
	fieldName := rnd.identGenerator.Field(svc.id)

	fmt.Fprintf(b, "func (c *%s) %s() (%s, error) {\n", rnd.containerName, getter, getterTypeStr)
	fmt.Fprintf(b, "\tif c.%s != nil {\n\t\treturn c.%s, nil\n\t}\n", fieldName, fieldName)
	fmt.Fprintf(b, "\tres, err := %s\n", rnd.getterBuildExpr(svc))
	fmt.Fprintf(b, "\tif err != nil {\n")
	fmt.Fprintf(b, "\t\treturn nil, err\n\t}\n")
	fmt.Fprintf(b, "\tc.%s = res\n", fieldName)
	fmt.Fprintf(b, "\treturn res, nil\n")
	b.WriteString("}\n\n")
	return nil
}

// sharedValueGetterRenderer renders a getter for shared services with value types.
// Uses a separate Init flag for caching.
type sharedValueGetterRenderer struct{}

func (g *sharedValueGetterRenderer) render(b *bytes.Buffer, rnd *ContainerRenderer, _ *GenContext, svc *serviceDef) error {
	getter := svc.privateGetterName
	getterTypeStr := rnd.importManager.typeString(svc.GetterType())
	fieldName := rnd.identGenerator.Field(svc.id)

	fmt.Fprintf(b, "func (c *%s) %s() (%s, error) {\n", rnd.containerName, getter, getterTypeStr)
	fmt.Fprintf(b, "\tvar zero %s\n", getterTypeStr)
	fmt.Fprintf(b, "\tif c.%sInit {\n", fieldName)
	fmt.Fprintf(b, "\t\treturn c.%s, nil\n", fieldName)
	fmt.Fprintf(b, "\t}\n")
	fmt.Fprintf(b, "\tres, err := %s\n", rnd.getterBuildExpr(svc))
	fmt.Fprintf(b, "\tif err != nil {\n")
	fmt.Fprintf(b, "\t\treturn zero, err\n\t}\n")
	fmt.Fprintf(b, "\tc.%s = res\n", fieldName)
	fmt.Fprintf(b, "\tc.%sInit = true\n", fieldName)
	fmt.Fprintf(b, "\treturn res, nil\n")
	b.WriteString("}\n\n")
	return nil
}

// nonSharedGetterRenderer renders a getter for non-shared (prototype) services.
// No caching - builds a new instance on every call.
type nonSharedGetterRenderer struct{}

func (g *nonSharedGetterRenderer) render(b *bytes.Buffer, rnd *ContainerRenderer, _ *GenContext, svc *serviceDef) error {
	getter := svc.privateGetterName
	getterTypeStr := rnd.importManager.typeString(svc.GetterType())

	fmt.Fprintf(b, "func (c *%s) %s() (%s, error) {\n", rnd.containerName, getter, getterTypeStr)
	fmt.Fprintf(b, "\treturn %s\n", rnd.getterBuildExpr(svc))
	b.WriteString("}\n\n")
	return nil
}
