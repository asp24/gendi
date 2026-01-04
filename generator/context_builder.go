package generator

import (
	di "github.com/asp24/gendi"
	"github.com/asp24/gendi/ir"
	"github.com/asp24/gendi/typeres"
)

// ContextBuilder builds the generation context using the IR layer.
type ContextBuilder struct {
	cfg     *di.Config
	options Options
	loader  *typeres.Resolver
}

// NewContextBuilder creates a new context builder.
func NewContextBuilder(cfg *di.Config, options Options) *ContextBuilder {
	return &ContextBuilder{
		cfg:     cfg,
		options: options,
	}
}

// Build executes all phases and returns the generation context and renderer.
func (b *ContextBuilder) Build() (*genContext, *Renderer, error) {
	if err := b.initTypeLoader(); err != nil {
		return nil, nil, err
	}

	// Build IR using the IR builder
	irBuilder := ir.NewBuilder(b.loader)
	container, err := irBuilder.Build(b.cfg)
	if err != nil {
		return nil, nil, err
	}

	// Convert IR to genContext for rendering
	return b.convertToGenContext(container)
}

func (b *ContextBuilder) initTypeLoader() error {
	paths, err := collectPackagePaths(b.cfg)
	if err != nil {
		return err
	}

	loader := typeres.NewResolver(b.options.ModuleRoot)
	if err := loader.LoadPackages(paths); err != nil {
		return err
	}
	b.loader = loader

	return nil
}

func (b *ContextBuilder) convertToGenContext(container *ir.Container) (*genContext, *Renderer, error) {
	imports := NewImportManager(b.options.OutputPkgPath)
	nameGen := newNameGenerator()

	services := make(map[string]*serviceDef)
	// Convert IR services to serviceDef
	for id, irSvc := range container.Services {
		svcDef := b.convertService(irSvc)
		services[id] = svcDef
	}

	ctx := &genContext{
		services:          services,
		orderedServiceIDs: container.ServiceIDsPostOrder(),
		tags:              container.Tags,
		outputPkgPath:     b.options.OutputPkgPath,
		paramGetters:      container.ParamGetters(),
	}

	rnd := NewRenderer(imports, nameGen, b.options.Container)

	return ctx, rnd, nil
}

func (b *ContextBuilder) convertService(irSvc *ir.Service) *serviceDef {
	svcDef := &serviceDef{
		id:       irSvc.ID,
		typeName: irSvc.Type,
		public:   irSvc.Public,
		shared:   irSvc.Shared,
		canError: irSvc.CanError,
		tags:     irSvc.Tags,
	}

	if cfg, ok := b.cfg.Services[irSvc.ID]; ok && cfg.Type != "" {
		if declType, err := b.loader.LookupType(cfg.Type); err == nil {
			svcDef.declaredType = declType
		}
	}

	if irSvc.IsAlias() {
		svcDef.aliasTarget = irSvc.Alias.ID
	}

	if irSvc.Constructor != nil {
		svcDef.constructor = b.convertConstructor(irSvc.Constructor)
	}

	return svcDef
}

func (b *ContextBuilder) convertConstructor(irCons *ir.Constructor) constructorDef {
	result := constructorDef{
		argDefs:      irCons.Args,
		typeArgs:     irCons.TypeArgs,
		params:       irCons.Params,
		result:       irCons.ResultType,
		returnsError: irCons.ReturnsError,
	}

	if irCons.Kind == ir.FuncConstructor {
		result.kind = "func"
		result.funcObj = irCons.Func

		return result
	}

	result.kind = "method"
	result.methodObj = irCons.Func
	if irCons.Receiver != nil {
		result.methodRecvID = irCons.Receiver.ID
	}

	return result
}
