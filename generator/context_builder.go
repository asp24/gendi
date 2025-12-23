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
	decoratorsByBase := make(map[string][]*serviceDef)
	baseByDecorator := make(map[string]string)
	buildCanError := make(map[string]bool)
	getterCanError := make(map[string]bool)
	paramGetters := make(map[string]string)

	// Convert IR services to serviceDef
	for id, irSvc := range container.Services {
		svcDef := b.convertService(irSvc)
		services[id] = svcDef
		buildCanError[id] = irSvc.BuildCanError
		getterCanError[id] = irSvc.CanError
	}

	// Build decorator mappings
	for id, irSvc := range container.Services {
		if irSvc.IsDecorator() {
			baseID := irSvc.Decorates.ID
			decoratorsByBase[baseID] = append(decoratorsByBase[baseID], services[id])
			baseByDecorator[id] = baseID
		}
	}

	// Sort decorators by priority (already sorted in IR, but keep order consistent)
	for baseID, irSvc := range container.Services {
		if len(irSvc.Decorators) > 0 {
			decs := make([]*serviceDef, len(irSvc.Decorators))
			for i, dec := range irSvc.Decorators {
				decs[i] = services[dec.ID]
			}
			decoratorsByBase[baseID] = decs
		}
	}

	// Collect parameter getter methods
	for name, param := range container.Parameters {
		method := param.GetterMethod()
		if method != "" {
			paramGetters[name] = method
		}
	}

	// Also collect parameter getters from service args
	for _, irSvc := range container.Services {
		if irSvc.Constructor == nil {
			continue
		}
		for _, arg := range irSvc.Constructor.Args {
			if arg.Kind == ir.ParamRefArg && arg.Parameter != nil {
				method := arg.Parameter.GetterMethod()
				if method != "" {
					paramGetters[arg.Parameter.Name] = method
				}
			}
		}
	}

	ctx := &genContext{
		services:          services,
		orderedServiceIDs: container.ServiceOrder,
		decoratorsByBase:  decoratorsByBase,
		baseByDecorator:   baseByDecorator,
		loader:            b.loader,
		imports:           imports,
		outputPkgPath:     b.loader.outputPkgPath,
		containerName:     b.options.Container,
		buildCanError:     buildCanError,
		getterCanError:    getterCanError,
		cfg:               b.cfg,
		paramGetters:      paramGetters,
	}

	return ctx, nil
}

func (b *ContextBuilder) convertService(irSvc *ir.Service) *serviceDef {
	svcDef := &serviceDef{
		id:                 irSvc.ID,
		cfg:                b.cfg.Services[irSvc.ID],
		typeName:           irSvc.Type,
		public:             irSvc.Public,
		shared:             irSvc.Shared,
		canError:           irSvc.CanError,
		decorationPriority: irSvc.Priority,
		isDecorator:        irSvc.IsDecorator(),
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
	if svc, ok := b.cfg.Services[svcID]; ok {
		cons.argDefs = svc.Constructor.Args
	}

	return cons
}

// convertConstructorArgs converts IR arguments back to di.Argument for render compatibility.
// This is a bridge until render is updated to use IR directly.
func (b *ContextBuilder) convertConstructorArgs(irArgs []*ir.Argument) []di.Argument {
	args := make([]di.Argument, len(irArgs))
	for i, irArg := range irArgs {
		switch irArg.Kind {
		case ir.ServiceRefArg:
			args[i] = di.Argument{Kind: di.ArgServiceRef, Value: irArg.Service.ID}
		case ir.InnerArg:
			args[i] = di.Argument{Kind: di.ArgInner}
		case ir.ParamRefArg:
			args[i] = di.Argument{Kind: di.ArgParam, Value: irArg.Parameter.Name}
		case ir.TaggedArg:
			args[i] = di.Argument{Kind: di.ArgTagged, Value: irArg.Tag.Name}
		case ir.LiteralArg:
			// For literals, we need the original yaml.Node which is in the config
			// The render will use the original config's args
		}
	}
	return args
}
