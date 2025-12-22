package generator

import (
	"errors"
	"fmt"
	"go/format"
	"go/types"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	di "github.com/asp24/gendi"
)

type Options struct {
	Out           string
	Package       string
	Container     string
	Strict        bool
	BuildTags     string
	Verbose       bool
	ModulePath    string
	ModuleRoot    string
	OutputPkgPath string
}

type Generator struct {
	cfg     *di.Config
	passes  []di.Pass
	options Options
}

func New(cfg *di.Config, opts Options, passes []di.Pass) *Generator {
	return &Generator{cfg: cfg, passes: passes, options: opts}
}

func (g *Generator) Generate() ([]byte, error) {
	if g.options.Strict {
		// strict is default; keep here for clarity
	}
	if g.options.Container == "" {
		g.options.Container = "Container"
	}

	for _, pass := range g.passes {
		if err := pass.Process(g.cfg); err != nil {
			return nil, fmt.Errorf("compiler pass %q failed: %w", pass.Name(), err)
		}
	}

	ctx, err := g.buildContext()
	if err != nil {
		return nil, err
	}

	code, err := g.render(ctx)
	if err != nil {
		return nil, err
	}

	formatted, err := format.Source(code)
	if err != nil {
		return nil, fmt.Errorf("format generated code: %w", err)
	}
	return formatted, nil
}

type genContext struct {
	services          map[string]*serviceDef
	orderedServiceIDs []string
	decoratorsByBase  map[string][]*serviceDef
	baseByDecorator   map[string]string
	loader            *typeLoader
	imports           *importManager
	outputPkgPath     string
	containerName     string
	hasShared         bool
	buildCanError     map[string]bool
	getterCanError    map[string]bool
	cfg               *di.Config
	paramGetters      map[string]string
}

type serviceDef struct {
	id                 string
	cfg                *di.Service
	typeName           types.Type
	constructor        constructorDef
	getterName         string
	privateGetterName  string
	public             bool
	shared             bool
	canError           bool
	decorates          string
	decorationPriority int
	isDecorator        bool
	aliasTarget        string
}

// IsAlias returns true if this service is an alias to another service.
func (s *serviceDef) IsAlias() bool {
	return s.cfg.Alias != ""
}

// HasConstructor returns true if this service defines a constructor.
func (s *serviceDef) HasConstructor() bool {
	return s.constructor.kind != ""
}

// Dependencies returns the service IDs this service depends on.
func (s *serviceDef) Dependencies(cfg *di.Config) ([]string, error) {
	return constructorDeps(s.id, s, cfg)
}

// ResolutionTracker tracks service resolution state and detects circular references.
type ResolutionTracker struct {
	resolving map[string]bool
	resolved  map[string]bool
}

// NewResolutionTracker creates a new resolution tracker.
func NewResolutionTracker() *ResolutionTracker {
	return &ResolutionTracker{
		resolving: make(map[string]bool),
		resolved:  make(map[string]bool),
	}
}

// IsResolved returns true if the service has been resolved.
func (r *ResolutionTracker) IsResolved(id string) bool {
	return r.resolved[id]
}

// StartResolving marks a service as being resolved and checks for cycles.
// Returns an error if the service is already being resolved (circular reference).
func (r *ResolutionTracker) StartResolving(id string) error {
	if r.resolving[id] {
		return fmt.Errorf("circular constructor reference at %q", id)
	}
	r.resolving[id] = true
	return nil
}

// FinishResolving marks a service as resolved.
func (r *ResolutionTracker) FinishResolving(id string) {
	r.resolving[id] = false
	r.resolved[id] = true
}

// resolveAliasService resolves an alias service's type from its target.
func resolveAliasService(svc *serviceDef, services map[string]*serviceDef, loader *typeLoader, resolveFunc func(string) error) error {
	if svc.cfg.Constructor.Func != "" || svc.cfg.Constructor.Method != "" || len(svc.cfg.Constructor.Args) > 0 {
		return fmt.Errorf("service %q alias cannot define constructor", svc.id)
	}
	if svc.cfg.Decorates != "" {
		return fmt.Errorf("service %q alias cannot be a decorator", svc.id)
	}

	if err := resolveFunc(svc.cfg.Alias); err != nil {
		return err
	}

	target := services[svc.cfg.Alias]
	if target == nil {
		return fmt.Errorf("service %q alias target %q not found", svc.id, svc.cfg.Alias)
	}

	svc.aliasTarget = svc.cfg.Alias
	svc.typeName = target.typeName

	if svc.cfg.Type != "" {
		declType, err := loader.lookupType(svc.cfg.Type)
		if err != nil {
			return fmt.Errorf("service %q type: %w", svc.id, err)
		}
		if !types.AssignableTo(svc.typeName, declType) {
			return fmt.Errorf("service %q type mismatch: expected %s, got %s", svc.id, loader.typeString(declType), loader.typeString(svc.typeName))
		}
	}
	return nil
}

// resolveConstructorService resolves a service's constructor and result type.
func resolveConstructorService(svc *serviceDef, services map[string]*serviceDef, loader *typeLoader, resolveFunc func(string) error) error {
	cons, err := resolveConstructor(svc.id, svc.cfg, loader, services, resolveFunc)
	if err != nil {
		return err
	}
	svc.constructor = cons
	svc.typeName = cons.result

	if svc.cfg.Type != "" {
		declType, err := loader.lookupType(svc.cfg.Type)
		if err != nil {
			return fmt.Errorf("service %q type: %w", svc.id, err)
		}
		if !types.AssignableTo(svc.typeName, declType) {
			return fmt.Errorf("service %q type mismatch: expected %s, got %s", svc.id, loader.typeString(declType), loader.typeString(svc.typeName))
		}
	}
	return nil
}

type constructorDef struct {
	kind         string // func|method
	funcObj      *types.Func
	methodObj    *types.Func
	methodRecvID string
	params       []types.Type
	result       types.Type
	returnsError bool
	argDefs      []di.Argument
}

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

func (g *Generator) buildContext() (*genContext, error) {
	builder := NewContextBuilder(g.cfg, g.options)
	return builder.Build()
}

func constructorDeps(id string, svc *serviceDef, cfg *di.Config) ([]string, error) {
	deps := []string{}
	if svc.aliasTarget != "" {
		return []string{svc.aliasTarget}, nil
	}
	cons := svc.constructor
	if cons.kind == "method" {
		deps = append(deps, cons.methodRecvID)
	}
	for _, arg := range cons.argDefs {
		switch arg.Kind {
		case di.ArgServiceRef:
			deps = append(deps, arg.Value)
		case di.ArgInner:
			if svc.decorates == "" {
				return nil, fmt.Errorf("service %q uses @.inner but is not a decorator", id)
			}
			deps = append(deps, svc.decorates)
		case di.ArgTagged:
			for sid, tagSvc := range cfg.Services {
				for _, t := range tagSvc.Tags {
					if t.Name == arg.Value {
						deps = append(deps, sid)
						break
					}
				}
			}
		}
	}
	return uniqueStrings(deps), nil
}

// CycleDetector detects circular dependencies in the service graph using DFS.
type CycleDetector struct {
	services map[string]*serviceDef
	cfg      *di.Config
	visited  map[string]bool
	stack    map[string]bool
}

// NewCycleDetector creates a new cycle detector for the given services.
func NewCycleDetector(services map[string]*serviceDef, cfg *di.Config) *CycleDetector {
	return &CycleDetector{
		services: services,
		cfg:      cfg,
		visited:  make(map[string]bool),
		stack:    make(map[string]bool),
	}
}

// Detect checks for circular dependencies and returns an error if found.
func (d *CycleDetector) Detect() error {
	for id := range d.services {
		if err := d.dfs(id, nil); err != nil {
			return err
		}
	}
	return nil
}

func (d *CycleDetector) dfs(id string, path []string) error {
	if d.stack[id] {
		cycle := append(path, id)
		return fmt.Errorf("circular dependency: %s", strings.Join(cycle, " -> "))
	}
	if d.visited[id] {
		return nil
	}
	d.visited[id] = true
	d.stack[id] = true

	deps, err := constructorDeps(id, d.services[id], d.cfg)
	if err != nil {
		return err
	}
	for _, dep := range deps {
		if err := d.dfs(dep, append(path, id)); err != nil {
			return err
		}
	}

	d.stack[id] = false
	return nil
}

// ErrorPropagationCalculator computes which service getters and builders can return errors.
type ErrorPropagationCalculator struct {
	cfg *di.Config
}

// NewErrorPropagationCalculator creates a new error propagation calculator.
func NewErrorPropagationCalculator(cfg *di.Config) *ErrorPropagationCalculator {
	return &ErrorPropagationCalculator{cfg: cfg}
}

// ErrorPropagationResult holds the computed error propagation maps.
type ErrorPropagationResult struct {
	BuildCanError  map[string]bool
	GetterCanError map[string]bool
}

// Calculate computes error propagation for the given services and decorator mappings.
func (c *ErrorPropagationCalculator) Calculate(services map[string]*serviceDef, decoratorsByBase map[string][]*serviceDef) ErrorPropagationResult {
	result := ErrorPropagationResult{
		BuildCanError:  make(map[string]bool),
		GetterCanError: make(map[string]bool),
	}

	// Initialize from constructors
	for id, svc := range services {
		result.GetterCanError[id] = svc.constructor.returnsError
		result.BuildCanError[id] = svc.constructor.returnsError
	}

	// Propagate errors until stable
	changed := true
	for changed {
		changed = false
		changed = c.propagateBuildErrors(services, &result) || changed
		changed = c.propagateGetterErrors(services, decoratorsByBase, &result) || changed
	}

	return result
}

func (c *ErrorPropagationCalculator) propagateBuildErrors(services map[string]*serviceDef, result *ErrorPropagationResult) bool {
	changed := false
	for id, svc := range services {
		can := svc.constructor.returnsError
		deps, _ := buildDeps(id, svc, c.cfg)
		for _, dep := range deps {
			if result.GetterCanError[dep] {
				can = true
			}
		}
		if result.BuildCanError[id] != can {
			result.BuildCanError[id] = can
			changed = true
		}
	}
	return changed
}

func (c *ErrorPropagationCalculator) propagateGetterErrors(services map[string]*serviceDef, decoratorsByBase map[string][]*serviceDef, result *ErrorPropagationResult) bool {
	changed := false
	for id, svc := range services {
		newVal := c.computeGetterCanError(id, svc, decoratorsByBase, result)
		if result.GetterCanError[id] != newVal {
			result.GetterCanError[id] = newVal
			changed = true
		}
	}
	return changed
}

func (c *ErrorPropagationCalculator) computeGetterCanError(id string, svc *serviceDef, decoratorsByBase map[string][]*serviceDef, result *ErrorPropagationResult) bool {
	if svc.isDecorator {
		return c.computeDecoratorGetterError(id, svc, decoratorsByBase, result)
	}
	if decs := decoratorsByBase[id]; len(decs) > 0 {
		return c.computeDecoratedServiceGetterError(id, decs, result)
	}
	return result.BuildCanError[id]
}

func (c *ErrorPropagationCalculator) computeDecoratorGetterError(id string, svc *serviceDef, decoratorsByBase map[string][]*serviceDef, result *ErrorPropagationResult) bool {
	base := svc.decorates
	if result.BuildCanError[base] {
		return true
	}
	decs := decoratorsByBase[base]
	for _, d := range decs {
		if result.BuildCanError[d.id] {
			return true
		}
		if d.id == id {
			break
		}
	}
	return false
}

func (c *ErrorPropagationCalculator) computeDecoratedServiceGetterError(id string, decs []*serviceDef, result *ErrorPropagationResult) bool {
	if result.BuildCanError[id] {
		return true
	}
	for _, d := range decs {
		if result.BuildCanError[d.id] {
			return true
		}
	}
	return false
}

// computeGetterErrors is a convenience function that wraps ErrorPropagationCalculator.
func computeGetterErrors(services map[string]*serviceDef, cfg *di.Config, decoratorsByBase map[string][]*serviceDef) (map[string]bool, map[string]bool) {
	calc := NewErrorPropagationCalculator(cfg)
	result := calc.Calculate(services, decoratorsByBase)
	return result.BuildCanError, result.GetterCanError
}

func buildDeps(id string, svc *serviceDef, cfg *di.Config) ([]string, error) {
	deps := []string{}
	if svc.aliasTarget != "" {
		return []string{svc.aliasTarget}, nil
	}
	cons := svc.constructor
	if cons.kind == "method" {
		deps = append(deps, cons.methodRecvID)
	}
	for _, arg := range cons.argDefs {
		switch arg.Kind {
		case di.ArgServiceRef:
			deps = append(deps, arg.Value)
		case di.ArgTagged:
			for sid, tagSvc := range cfg.Services {
				for _, t := range tagSvc.Tags {
					if t.Name == arg.Value {
						deps = append(deps, sid)
						break
					}
				}
			}
		case di.ArgInner:
			// inner is provided by decorator chain; its errors are handled there.
		}
	}
	return uniqueStrings(deps), nil
}

func resolveConstructor(id string, svc *di.Service, loader *typeLoader, services map[string]*serviceDef, resolve func(string) error) (constructorDef, error) {
	cons := svc.Constructor
	if cons.Func == "" && cons.Method == "" {
		return constructorDef{}, fmt.Errorf("service %q missing constructor", id)
	}
	if cons.Func != "" && cons.Method != "" {
		return constructorDef{}, fmt.Errorf("service %q has both func and method constructors", id)
	}

	if cons.Func != "" {
		pkgPath, name, err := splitPkgSymbol(cons.Func)
		if err != nil {
			return constructorDef{}, fmt.Errorf("service %q constructor.func: %w", id, err)
		}
		obj, err := loader.lookupFunc(pkgPath, name)
		if err != nil {
			return constructorDef{}, fmt.Errorf("service %q constructor.func: %w", id, err)
		}
		sig := obj.Type().(*types.Signature)
		resType, returnsErr, err := validateConstructorSignature(sig)
		if err != nil {
			return constructorDef{}, fmt.Errorf("service %q constructor.func: %w", id, err)
		}
		params := signatureParams(sig)
		return constructorDef{
			kind:         "func",
			funcObj:      obj,
			params:       params,
			result:       resType,
			returnsError: returnsErr,
			argDefs:      cons.Args,
		}, nil
	}

	// method
	methodRef := cons.Method
	if !strings.HasPrefix(methodRef, "@") {
		return constructorDef{}, fmt.Errorf("service %q constructor.method must start with @", id)
	}
	methodRef = methodRef[1:]
	parts := strings.Split(methodRef, ".")
	if len(parts) < 2 {
		return constructorDef{}, fmt.Errorf("service %q constructor.method invalid format", id)
	}
	methodName := parts[len(parts)-1]
	recvID := strings.Join(parts[:len(parts)-1], ".")
	if recvID == "" || methodName == "" {
		return constructorDef{}, fmt.Errorf("service %q constructor.method invalid format", id)
	}
	if err := resolve(recvID); err != nil {
		return constructorDef{}, err
	}
	recvSvc := services[recvID]
	if recvSvc == nil {
		return constructorDef{}, fmt.Errorf("service %q constructor.method unknown receiver service %q", id, recvID)
	}
	meth, err := loader.lookupMethod(recvSvc.typeName, methodName)
	if err != nil {
		return constructorDef{}, fmt.Errorf("service %q constructor.method: %w", id, err)
	}
	msig := meth.Type().(*types.Signature)
	resType, returnsErr, err := validateConstructorSignature(msig)
	if err != nil {
		return constructorDef{}, fmt.Errorf("service %q constructor.method: %w", id, err)
	}
	params := signatureParams(msig)
	return constructorDef{
		kind:         "method",
		methodObj:    meth,
		methodRecvID: recvID,
		params:       params,
		result:       resType,
		returnsError: returnsErr,
		argDefs:      cons.Args,
	}, nil
}

func signatureParams(sig *types.Signature) []types.Type {
	params := []types.Type{}
	for i := 0; i < sig.Params().Len(); i++ {
		params = append(params, sig.Params().At(i).Type())
	}
	return params
}

func validateConstructorSignature(sig *types.Signature) (types.Type, bool, error) {
	res := sig.Results()
	if res.Len() == 0 || res.Len() > 2 {
		return nil, false, fmt.Errorf("constructor must return T or (T, error)")
	}
	resType := res.At(0).Type()
	returnsErr := false
	if res.Len() == 2 {
		errType := res.At(1).Type()
		if !isErrorType(errType) {
			return nil, false, fmt.Errorf("second return value must be error")
		}
		returnsErr = true
	}
	if err := validateServiceType(resType); err != nil {
		return nil, false, err
	}
	return resType, returnsErr, nil
}

func validateServiceType(t types.Type) error {
	switch tt := t.(type) {
	case *types.Pointer:
		return validateServiceType(tt.Elem())
	case *types.Named:
		//if _, ok := tt.Underlying().(*types.Interface); ok {
		//	return fmt.Errorf("service type must not be interface")
		//}
		return nil
	case *types.TypeParam:
		return fmt.Errorf("service type must not be type parameter")
	default:
		return fmt.Errorf("service type must be a named concrete type")
	}
}

func isErrorType(t types.Type) bool {
	return types.Identical(t, types.Universe.Lookup("error").Type())
}

func validateArgs(id string, svc *serviceDef, services map[string]*serviceDef, cfg *di.Config, loader *typeLoader, decoratorsByBase map[string][]*serviceDef, canError map[string]bool) error {
	cons := svc.constructor
	params := cons.params
	if len(cons.argDefs) != len(params) {
		return fmt.Errorf("service %q constructor args count mismatch: expected %d got %d", id, len(params), len(cons.argDefs))
	}
	for i, arg := range cons.argDefs {
		paramType := params[i]
		switch arg.Kind {
		case di.ArgServiceRef:
			dep := services[arg.Value]
			if dep == nil {
				return fmt.Errorf("service %q arg[%d]: unknown service %q", id, i, arg.Value)
			}
			depType := getterType(dep, services, decoratorsByBase)
			if !types.AssignableTo(depType, paramType) {
				return fmt.Errorf("service %q arg[%d]: expected %s, got %s", id, i, loader.typeString(paramType), loader.typeString(depType))
			}
		case di.ArgInner:
			if svc.decorates == "" {
				return fmt.Errorf("service %q arg[%d]: @.inner used outside decorator", id, i)
			}
			baseSvc := services[svc.decorates]
			if baseSvc == nil {
				return fmt.Errorf("service %q arg[%d]: unknown base service %q", id, i, svc.decorates)
			}
			innerType := baseSvc.typeName
			if baseSvc.cfg.Type != "" {
				declType, err := loader.lookupType(baseSvc.cfg.Type)
				if err != nil {
					return fmt.Errorf("service %q arg[%d]: base %q type: %w", id, i, svc.decorates, err)
				}
				innerType = declType
			}
			if !types.AssignableTo(innerType, paramType) {
				return fmt.Errorf("service %q arg[%d]: expected %s, got %s", id, i, loader.typeString(paramType), loader.typeString(innerType))
			}
		case di.ArgParam:
			param, ok := cfg.Parameters[arg.Value]
			if ok {
				if param.Type == "" {
					return fmt.Errorf("service %q arg[%d]: parameter %q missing type", id, i, arg.Value)
				}
				paramDefType, err := loader.lookupType(param.Type)
				if err != nil {
					return fmt.Errorf("service %q arg[%d]: parameter %q type: %w", id, i, arg.Value, err)
				}
				if _, err := paramGetterMethod(paramDefType); err != nil {
					return fmt.Errorf("service %q arg[%d]: parameter %q type: %w", id, i, arg.Value, err)
				}
				if !types.AssignableTo(paramDefType, paramType) {
					return fmt.Errorf("service %q arg[%d]: parameter %q expected %s, got %s", id, i, arg.Value, loader.typeString(paramType), loader.typeString(paramDefType))
				}
				break
			}
			if _, err := paramGetterMethod(paramType); err != nil {
				return fmt.Errorf("service %q arg[%d]: parameter %q type: %w", id, i, arg.Value, err)
			}
		case di.ArgTagged:
			tag, ok := cfg.Tags[arg.Value]
			if !ok {
				return fmt.Errorf("service %q arg[%d]: unknown tag %q", id, i, arg.Value)
			}
			if tag.ElementType == "" {
				return fmt.Errorf("service %q arg[%d]: tag %q missing element_type", id, i, arg.Value)
			}
			elemType, err := loader.lookupType(tag.ElementType)
			if err != nil {
				return fmt.Errorf("service %q arg[%d]: tag %q element_type: %w", id, i, arg.Value, err)
			}
			sliceType := types.NewSlice(elemType)
			if !types.AssignableTo(sliceType, paramType) {
				return fmt.Errorf("service %q arg[%d]: expected %s, got %s", id, i, loader.typeString(paramType), loader.typeString(sliceType))
			}
		default:
			if isTimeDuration(paramType) {
				if _, err := durationLiteral(arg.Literal); err != nil {
					return fmt.Errorf("service %q arg[%d]: %w", id, i, err)
				}
				break
			}
			litType, err := literalType(arg.Literal)
			if err != nil {
				return fmt.Errorf("service %q arg[%d]: %w", id, i, err)
			}
			if !types.AssignableTo(litType, paramType) {
				return fmt.Errorf("service %q arg[%d]: expected %s, got %s", id, i, loader.typeString(paramType), loader.typeString(litType))
			}
		}
	}

	// Ensure error propagation is explicit.
	// if !svc.constructor.returnsError && svc.canError {
	// 	return fmt.Errorf("service %q depends on error-returning services but constructor does not return error", id)
	// }
	return nil
}

func literalType(node yaml.Node) (types.Type, error) {
	switch node.Tag {
	case "!!str":
		return types.Typ[types.UntypedString], nil
	case "!!int":
		return types.Typ[types.UntypedInt], nil
	case "!!bool":
		return types.Typ[types.UntypedBool], nil
	case "!!float":
		return types.Typ[types.UntypedFloat], nil
	case "!!null":
		return types.Typ[types.UntypedNil], nil
	default:
		return nil, fmt.Errorf("unsupported literal type %q", node.Tag)
	}
}

func getterType(svc *serviceDef, services map[string]*serviceDef, decoratorsByBase map[string][]*serviceDef) types.Type {
	if svc.aliasTarget != "" {
		if target := services[svc.aliasTarget]; target != nil {
			return getterType(target, services, decoratorsByBase)
		}
	}
	if svc.decorates != "" {
		return svc.typeName
	}
	if decs := decoratorsByBase[svc.id]; len(decs) > 0 {
		return decs[len(decs)-1].typeName
	}
	return svc.typeName
}

func splitPkgSymbol(s string) (string, string, error) {
	idx := strings.LastIndex(s, ".")
	if idx <= 0 || idx == len(s)-1 {
		return "", "", fmt.Errorf("invalid qualified name %q", s)
	}
	return s[:idx], s[idx+1:], nil
}

func paramGetterMethod(t types.Type) (string, error) {
	switch {
	case types.Identical(t, types.Typ[types.String]):
		return "GetString", nil
	case types.Identical(t, types.Typ[types.Int]):
		return "GetInt", nil
	case types.Identical(t, types.Typ[types.Bool]):
		return "GetBool", nil
	case types.Identical(t, types.Typ[types.Float64]):
		return "GetFloat", nil
	case isTimeDuration(t):
		return "GetDuration", nil
	default:
		return "", fmt.Errorf("unsupported parameter type %s", types.TypeString(t, nil))
	}
}

func collectPackagePaths(cfg *di.Config) ([]string, error) {
	seen := map[string]bool{}
	add := func(path string) {
		if path != "" {
			seen[path] = true
		}
	}

	for _, svc := range cfg.Services {
		if svc.Constructor.Func != "" {
			pkg, _, err := splitPkgSymbol(svc.Constructor.Func)
			if err != nil {
				return nil, err
			}
			add(pkg)
		}
		if svc.Type != "" {
			pkg, err := typePkgPath(svc.Type)
			if err != nil {
				return nil, err
			}
			add(pkg)
		}
	}
	for _, param := range cfg.Parameters {
		if param.Type != "" {
			pkg, err := typePkgPath(param.Type)
			if err != nil {
				return nil, err
			}
			add(pkg)
		}
	}
	for _, tag := range cfg.Tags {
		if tag.ElementType != "" {
			pkg, err := typePkgPath(tag.ElementType)
			if err != nil {
				return nil, err
			}
			add(pkg)
		}
	}

	out := make([]string, 0, len(seen))
	for path := range seen {
		out = append(out, path)
	}
	sort.Strings(out)
	return out, nil
}

func typePkgPath(typeStr string) (string, error) {
	t := strings.TrimPrefix(typeStr, "*")
	if !strings.Contains(t, ".") {
		return "", nil
	}
	pkg, _, err := splitPkgSymbol(t)
	if err != nil {
		return "", err
	}
	return pkg, nil
}

func uniqueStrings(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
