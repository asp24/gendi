package generator

import (
	di "github.com/asp24/gendi"
	"github.com/asp24/gendi/ir"
	"github.com/asp24/gendi/typeres"
)

// IRConverter builds the generation context using the IR layer.
type IRConverter struct {
	options      Options
	typeResolver *typeres.Resolver
}

func NewIRConverter(typeResolver *typeres.Resolver, options Options) *IRConverter {
	return &IRConverter{
		typeResolver: typeResolver,
		options:      options,
	}
}

func (b *IRConverter) convertConstructor(irCons *ir.Constructor) constructorDef {
	result := constructorDef{
		argDefs:      irCons.Args,
		typeArgs:     irCons.TypeArgs,
		params:       irCons.Params,
		result:       irCons.ResultType,
		returnsError: irCons.ReturnsError,
	}

	result.funcObj = irCons.Func

	if irCons.Kind == ir.FuncConstructor {
		result.kind = "func"
		return result
	}

	result.kind = "method"
	if irCons.Receiver != nil {
		result.methodRecvID = irCons.Receiver.ID
	}

	return result
}

func (b *IRConverter) convertService(irSvc *ir.Service, cfg *di.Config) *serviceDef {
	svcDef := &serviceDef{
		id:       irSvc.ID,
		typeName: irSvc.Type,
		public:   irSvc.Public,
		shared:   irSvc.Shared,
	}

	if irSvc.IsAlias() {
		svcDef.aliasTarget = irSvc.Alias.ID
	}

	if irSvc.Constructor != nil {
		svcDef.constructor = b.convertConstructor(irSvc.Constructor)
	}

	return svcDef
}

func (b *IRConverter) convertToGenContext(irContainer *ir.Container, cfg *di.Config) (*genContext, error) {
	services := make(map[string]*serviceDef)
	// Convert IR services to serviceDef
	for id, irSvc := range irContainer.Services {
		svcDef := b.convertService(irSvc, cfg)
		services[id] = svcDef
	}

	ctx := &genContext{
		services:          services,
		orderedServiceIDs: irContainer.ServiceIDsPostOrder(),
		outputPkgPath:     b.options.OutputPkgPath,
		paramGetters:      irContainer.ParamGetters(),
	}

	return ctx, nil
}

// Convert executes all phases and returns the generation context and renderer.
func (b *IRConverter) Convert(irContainer *ir.Container, cfg *di.Config) (*genContext, error) {
	// Convert IR to genContext for rendering
	return b.convertToGenContext(irContainer, cfg)
}
