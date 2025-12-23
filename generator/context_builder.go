package generator

import (
	di "github.com/asp24/gendi"
	"github.com/asp24/gendi/ir"
)

// ContextBuilder builds the generation context using the IR layer.
type ContextBuilder struct {
	cfg     *di.Config
	options Options
	loader  *TypeLoader
}

// NewContextBuilder creates a new context builder.
func NewContextBuilder(cfg *di.Config, options Options) *ContextBuilder {
	return &ContextBuilder{
		cfg:     cfg,
		options: options,
	}
}

// Build executes all phases and returns the generation context.
func (b *ContextBuilder) Build() (*genContext, error) {
	if err := b.initTypeLoader(); err != nil {
		return nil, err
	}

	// Build IR using the IR builder
	irBuilder := ir.NewBuilder(b.cfg, b.loader)
	container, err := irBuilder.Build()
	if err != nil {
		return nil, err
	}

	// Convert IR to genContext for rendering
	return b.convertToGenContext(container)
}

func (b *ContextBuilder) initTypeLoader() error {
	loader, err := NewTypeLoader(b.options)
	if err != nil {
		return err
	}
	paths, err := collectPackagePaths(b.cfg)
	if err != nil {
		return err
	}
	if err := loader.loadPackages(paths); err != nil {
		return err
	}
	b.loader = loader
	return nil
}

func (b *ContextBuilder) convertToGenContext(container *ir.Container) (*genContext, error) {
	imports := NewImportManager(b.loader.outputPkgPath)

	services := make(map[string]*serviceDef)
	// Convert IR services to serviceDef
	for id, irSvc := range container.Services {
		svcDef := b.convertService(irSvc)
		services[id] = svcDef
	}

	decoratorsByBase := make(map[string][]*serviceDef)
	for baseID, decs := range container.DecoratorsByBase() {
		items := make([]*serviceDef, len(decs))
		for i, dec := range decs {
			items[i] = services[dec.ID]
		}
		decoratorsByBase[baseID] = items
	}

	baseByDecorator := container.BaseByDecorator()
	paramGetters := container.ParamGetters()

	ctx := &genContext{
		services:          services,
		orderedServiceIDs: container.ServiceOrder,
		decoratorsByBase:  decoratorsByBase,
		baseByDecorator:   baseByDecorator,
		tags:              container.Tags,
		loader:            b.loader,
		imports:           imports,
		outputPkgPath:     b.loader.outputPkgPath,
		containerName:     b.options.Container,
		paramGetters:      paramGetters,
	}

	return ctx, nil
}

func (b *ContextBuilder) convertService(irSvc *ir.Service) *serviceDef {
	svcDef := &serviceDef{
		id:                 irSvc.ID,
		typeName:           irSvc.Type,
		public:             irSvc.Public,
		shared:             irSvc.Shared,
		canError:           irSvc.CanError,
		decorationPriority: irSvc.Priority,
		isDecorator:        irSvc.IsDecorator(),
		tags:               irSvc.Tags,
	}

	if cfg := b.cfg.Services[irSvc.ID]; cfg != nil && cfg.Type != "" {
		if declType, err := b.loader.LookupType(cfg.Type); err == nil {
			svcDef.declaredType = declType
		}
	}

	if irSvc.IsAlias() {
		svcDef.aliasTarget = irSvc.Alias.ID
	}

	if irSvc.IsDecorator() {
		svcDef.decorates = irSvc.Decorates.ID
	}

	if irSvc.Constructor != nil {
		svcDef.constructor = b.convertConstructor(irSvc.ID, irSvc.Constructor)
	}

	return svcDef
}

func (b *ContextBuilder) convertConstructor(svcID string, irCons *ir.Constructor) constructorDef {
	cons := constructorDef{
		funcObj:      irCons.Func,
		params:       irCons.Params,
		result:       irCons.ResultType,
		returnsError: irCons.ReturnsError,
	}

	if irCons.Kind == ir.FuncConstructor {
		cons.kind = "func"
		cons.funcObj = irCons.Func
	} else {
		cons.kind = "method"
		cons.methodObj = irCons.Func
		if irCons.Receiver != nil {
			cons.methodRecvID = irCons.Receiver.ID
		}
	}

	// Get arg definitions from original config
	cons.argDefs = irCons.Args

	return cons
}
