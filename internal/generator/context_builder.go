package generator

import (
	"errors"
	"fmt"
	"go/types"
	"sort"

	di "github.com/asp24/gendi"
)

// ContextBuilder builds the generation context in phases.
type ContextBuilder struct {
	cfg              *di.Config
	options          Options
	loader           *typeLoader
	services         map[string]*serviceDef
	order            []string
	decoratorsByBase map[string][]*serviceDef
	baseByDecorator  map[string]string
	buildCanError    map[string]bool
	getterCanError   map[string]bool
	paramGetters     map[string]string
}

// NewContextBuilder creates a new context builder.
func NewContextBuilder(cfg *di.Config, options Options) *ContextBuilder {
	return &ContextBuilder{
		cfg:              cfg,
		options:          options,
		services:         make(map[string]*serviceDef),
		decoratorsByBase: make(map[string][]*serviceDef),
		baseByDecorator:  make(map[string]string),
		paramGetters:     make(map[string]string),
	}
}

// Build executes all phases and returns the generation context.
func (b *ContextBuilder) Build() (*genContext, error) {
	if b.cfg.Services == nil {
		return nil, errors.New("no services defined")
	}

	if err := b.initTypeLoader(); err != nil {
		return nil, err
	}
	if err := b.initServices(); err != nil {
		return nil, err
	}
	if err := b.resolveConstructors(); err != nil {
		return nil, err
	}
	if err := b.validatePublicServices(); err != nil {
		return nil, err
	}
	if err := b.validateParameters(); err != nil {
		return nil, err
	}
	if err := b.buildDecoratorMappings(); err != nil {
		return nil, err
	}
	b.computeErrorPropagation()
	if err := b.collectParameterGetters(); err != nil {
		return nil, err
	}
	if err := b.validateArguments(); err != nil {
		return nil, err
	}
	if b.options.Strict {
		if err := b.detectCycles(); err != nil {
			return nil, err
		}
	}

	return b.buildResult()
}

func (b *ContextBuilder) initTypeLoader() error {
	loader, err := newTypeLoader(b.options)
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

func (b *ContextBuilder) initServices() error {
	b.order = make([]string, 0, len(b.cfg.Services))
	for id, svc := range b.cfg.Services {
		b.order = append(b.order, id)
		shared := true
		if svc.Shared != nil {
			shared = *svc.Shared
		}
		if svc.Alias != "" {
			shared = false
		}
		b.services[id] = &serviceDef{
			id:                 id,
			cfg:                svc,
			shared:             shared,
			public:             svc.Public,
			decorates:          svc.Decorates,
			decorationPriority: svc.DecorationPriority,
			isDecorator:        svc.Decorates != "",
		}
	}
	sort.Strings(b.order)
	return nil
}

func (b *ContextBuilder) resolveConstructors() error {
	tracker := NewResolutionTracker()
	var resolveService func(id string) error
	resolveService = func(id string) error {
		if tracker.IsResolved(id) {
			return nil
		}
		if err := tracker.StartResolving(id); err != nil {
			return err
		}
		svc := b.services[id]
		if svc == nil {
			return fmt.Errorf("unknown service %q", id)
		}
		var resolveErr error
		if svc.IsAlias() {
			resolveErr = resolveAliasService(svc, b.services, b.loader, resolveService)
		} else {
			resolveErr = resolveConstructorService(svc, b.services, b.loader, resolveService)
		}
		tracker.FinishResolving(id)
		return resolveErr
	}

	for _, id := range b.order {
		if err := resolveService(id); err != nil {
			return err
		}
	}
	return nil
}

func (b *ContextBuilder) validatePublicServices() error {
	for _, svc := range b.services {
		if svc.public {
			return nil
		}
	}
	return errors.New("at least one public service is required")
}

func (b *ContextBuilder) validateParameters() error {
	for name, param := range b.cfg.Parameters {
		if param.Type == "" {
			return fmt.Errorf("parameter %q missing type", name)
		}
		if param.Value.Kind == 0 {
			return fmt.Errorf("parameter %q missing value", name)
		}
		paramType, err := b.loader.lookupType(param.Type)
		if err != nil {
			return fmt.Errorf("parameter %q type: %w", name, err)
		}
		if _, err := paramGetterMethod(paramType); err != nil {
			return fmt.Errorf("parameter %q type: %w", name, err)
		}
		if isTimeDuration(paramType) {
			if _, err := durationLiteral(param.Value); err != nil {
				return fmt.Errorf("parameter %q value: %w", name, err)
			}
			continue
		}
		litType, err := literalType(param.Value)
		if err != nil {
			return fmt.Errorf("parameter %q value: %w", name, err)
		}
		if !types.AssignableTo(litType, paramType) {
			return fmt.Errorf("parameter %q value: expected %s, got %s", name, b.loader.typeString(paramType), b.loader.typeString(litType))
		}
	}
	return nil
}

func (b *ContextBuilder) buildDecoratorMappings() error {
	for _, svc := range b.services {
		if !svc.isDecorator {
			continue
		}
		base := svc.decorates
		baseSvc := b.services[base]
		if baseSvc == nil {
			return fmt.Errorf("decorator %q decorates unknown service %q", svc.id, base)
		}
		baseType := baseSvc.typeName
		if baseSvc.cfg.Type != "" {
			declType, err := b.loader.lookupType(baseSvc.cfg.Type)
			if err != nil {
				return fmt.Errorf("decorator %q base %q type: %w", svc.id, base, err)
			}
			baseType = declType
		}
		if !types.AssignableTo(svc.typeName, baseType) {
			return fmt.Errorf("decorator %q type %s not assignable to %s", svc.id, b.loader.typeString(svc.typeName), b.loader.typeString(baseType))
		}
		b.decoratorsByBase[base] = append(b.decoratorsByBase[base], svc)
		b.baseByDecorator[svc.id] = base
	}

	for base, decs := range b.decoratorsByBase {
		sort.Slice(decs, func(i, j int) bool { return decs[i].decorationPriority < decs[j].decorationPriority })
		b.decoratorsByBase[base] = decs
	}
	return nil
}

func (b *ContextBuilder) computeErrorPropagation() {
	b.buildCanError, b.getterCanError = computeGetterErrors(b.services, b.cfg, b.decoratorsByBase)
	for id, svc := range b.services {
		svc.canError = b.getterCanError[id]
	}
}

func (b *ContextBuilder) collectParameterGetters() error {
	for _, id := range b.order {
		svc := b.services[id]
		cons := svc.constructor
		for i, arg := range cons.argDefs {
			if arg.Kind != di.ArgParam {
				continue
			}
			if i >= len(cons.params) {
				continue
			}
			method, err := paramGetterMethod(cons.params[i])
			if err != nil {
				return fmt.Errorf("service %q arg[%d]: parameter %q type: %w", id, i, arg.Value, err)
			}
			if existing, ok := b.paramGetters[arg.Value]; ok && existing != method {
				return fmt.Errorf("parameter %q used with conflicting types", arg.Value)
			}
			b.paramGetters[arg.Value] = method
		}
	}
	return nil
}

func (b *ContextBuilder) validateArguments() error {
	for _, id := range b.order {
		svc := b.services[id]
		if err := validateArgs(id, svc, b.services, b.cfg, b.loader, b.decoratorsByBase, b.getterCanError); err != nil {
			return err
		}
	}
	return nil
}

func (b *ContextBuilder) detectCycles() error {
	detector := NewCycleDetector(b.services, b.cfg)
	return detector.Detect()
}

func (b *ContextBuilder) buildResult() (*genContext, error) {
	imports := newImportManager(b.loader.outputPkgPath)
	ctx := &genContext{
		services:          b.services,
		orderedServiceIDs: b.order,
		decoratorsByBase:  b.decoratorsByBase,
		baseByDecorator:   b.baseByDecorator,
		loader:            b.loader,
		imports:           imports,
		outputPkgPath:     b.loader.outputPkgPath,
		containerName:     b.options.Container,
		buildCanError:     b.buildCanError,
		getterCanError:    b.getterCanError,
		cfg:               b.cfg,
		paramGetters:      b.paramGetters,
	}

	for _, svc := range b.services {
		if svc.shared {
			ctx.hasShared = true
			break
		}
	}

	for name, param := range b.cfg.Parameters {
		paramType, err := b.loader.lookupType(param.Type)
		if err != nil {
			return nil, fmt.Errorf("parameter %q type: %w", name, err)
		}
		method, err := paramGetterMethod(paramType)
		if err != nil {
			return nil, fmt.Errorf("parameter %q type: %w", name, err)
		}
		if existing, ok := ctx.paramGetters[name]; ok && existing != method {
			return nil, fmt.Errorf("parameter %q used with conflicting types", name)
		}
		ctx.paramGetters[name] = method
	}

	return ctx, nil
}
