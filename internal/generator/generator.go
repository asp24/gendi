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

func (g *Generator) buildContext() (*genContext, error) {
	cfg := g.cfg
	if cfg.Services == nil {
		return nil, errors.New("no services defined")
	}

	loader, err := newTypeLoader(g.options)
	if err != nil {
		return nil, err
	}
	paths, err := collectPackagePaths(cfg)
	if err != nil {
		return nil, err
	}
	if err := loader.loadPackages(paths); err != nil {
		return nil, err
	}

	services := map[string]*serviceDef{}
	order := make([]string, 0, len(cfg.Services))
	for id, svc := range cfg.Services {
		order = append(order, id)
		shared := true
		if svc.Shared != nil {
			shared = *svc.Shared
		}
		services[id] = &serviceDef{
			id:                 id,
			cfg:                svc,
			shared:             shared,
			public:             svc.Public,
			decorates:          svc.Decorates,
			decorationPriority: svc.DecorationPriority,
			isDecorator:        svc.Decorates != "",
		}
	}
	sort.Strings(order)

	// Resolve constructor types.
	resolving := map[string]bool{}
	resolved := map[string]bool{}
	var resolveService func(id string) error
	resolveService = func(id string) error {
		if resolved[id] {
			return nil
		}
		if resolving[id] {
			return fmt.Errorf("circular constructor reference at %q", id)
		}
		svc := services[id]
		if svc == nil {
			return fmt.Errorf("unknown service %q", id)
		}
		resolving[id] = true
		cons, err := resolveConstructor(id, svc.cfg, loader, services, resolveService)
		if err != nil {
			return err
		}
		svc.constructor = cons
		svc.typeName = cons.result
		if svc.cfg.Type != "" {
			declType, err := loader.lookupType(svc.cfg.Type)
			if err != nil {
				return fmt.Errorf("service %q type: %w", id, err)
			}
			if !types.Identical(declType, svc.typeName) {
				return fmt.Errorf("service %q type mismatch: expected %s, got %s", id, loader.typeString(declType), loader.typeString(svc.typeName))
			}
		}
		resolving[id] = false
		resolved[id] = true
		return nil
	}

	for _, id := range order {
		if err := resolveService(id); err != nil {
			return nil, err
		}
	}
	hasPublic := false
	for _, svc := range services {
		if svc.public {
			hasPublic = true
			break
		}
	}
	if !hasPublic {
		return nil, errors.New("at least one public service is required")
	}

	for name, param := range cfg.Parameters {
		if param.Type == "" {
			return nil, fmt.Errorf("parameter %q missing type", name)
		}
		if param.Value.Kind == 0 {
			return nil, fmt.Errorf("parameter %q missing value", name)
		}
		if _, err := loader.lookupType(param.Type); err != nil {
			return nil, fmt.Errorf("parameter %q type: %w", name, err)
		}
		paramType, err := loader.lookupType(param.Type)
		if err != nil {
			return nil, fmt.Errorf("parameter %q type: %w", name, err)
		}
		if _, err := paramGetterMethod(paramType); err != nil {
			return nil, fmt.Errorf("parameter %q type: %w", name, err)
		}
		litType, err := literalType(param.Value)
		if err != nil {
			return nil, fmt.Errorf("parameter %q value: %w", name, err)
		}
		if !types.AssignableTo(litType, paramType) {
			return nil, fmt.Errorf("parameter %q value: expected %s, got %s", name, loader.typeString(paramType), loader.typeString(litType))
		}
	}

	// Decorators mapping and type checks.
	decoratorsByBase := map[string][]*serviceDef{}
	baseByDecorator := map[string]string{}
	for _, svc := range services {
		if svc.isDecorator {
			base := svc.decorates
			baseSvc := services[base]
			if baseSvc == nil {
				return nil, fmt.Errorf("decorator %q decorates unknown service %q", svc.id, base)
			}
			if !types.AssignableTo(svc.typeName, baseSvc.typeName) {
				return nil, fmt.Errorf("decorator %q type %s not assignable to %s", svc.id, loader.typeString(svc.typeName), loader.typeString(baseSvc.typeName))
			}
			decoratorsByBase[base] = append(decoratorsByBase[base], svc)
			baseByDecorator[svc.id] = base
		}
	}
	for base, decs := range decoratorsByBase {
		sort.Slice(decs, func(i, j int) bool { return decs[i].decorationPriority < decs[j].decorationPriority })
		decoratorsByBase[base] = decs
	}

	// Compute error propagation for build and getters.
	buildCanError, getterCanError := computeGetterErrors(services, cfg, decoratorsByBase)
	for id, svc := range services {
		svc.canError = getterCanError[id]
	}

	// Collect parameter getters from usage.
	paramGetters := map[string]string{}
	for _, id := range order {
		svc := services[id]
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
				return nil, fmt.Errorf("service %q arg[%d]: parameter %q type: %w", id, i, arg.Value, err)
			}
			if existing, ok := paramGetters[arg.Value]; ok && existing != method {
				return nil, fmt.Errorf("parameter %q used with conflicting types", arg.Value)
			}
			paramGetters[arg.Value] = method
		}
	}

	// Validate argument types.
	for _, id := range order {
		svc := services[id]
		if err := validateArgs(id, svc, services, cfg, loader, decoratorsByBase, getterCanError); err != nil {
			return nil, err
		}
	}

	// Detect cycles.
	if g.options.Strict {
		if err := detectCycles(services, cfg); err != nil {
			return nil, err
		}
	}

	imports := newImportManager(loader.outputPkgPath)
	ctx := &genContext{
		services:          services,
		orderedServiceIDs: order,
		decoratorsByBase:  decoratorsByBase,
		baseByDecorator:   baseByDecorator,
		loader:            loader,
		imports:           imports,
		outputPkgPath:     loader.outputPkgPath,
		containerName:     g.options.Container,
		buildCanError:     buildCanError,
		getterCanError:    getterCanError,
		cfg:               g.cfg,
		paramGetters:      paramGetters,
	}

	for _, svc := range services {
		if svc.shared {
			ctx.hasShared = true
		}
	}
	for name, param := range cfg.Parameters {
		paramType, err := loader.lookupType(param.Type)
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

func constructorDeps(id string, svc *serviceDef, cfg *di.Config) ([]string, error) {
	deps := []string{}
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

func detectCycles(services map[string]*serviceDef, cfg *di.Config) error {
	visited := map[string]bool{}
	stack := map[string]bool{}
	var dfs func(id string, path []string) error
	dfs = func(id string, path []string) error {
		if stack[id] {
			cycle := append(path, id)
			return fmt.Errorf("circular dependency: %s", strings.Join(cycle, " -> "))
		}
		if visited[id] {
			return nil
		}
		visited[id] = true
		stack[id] = true
		deps, err := constructorDeps(id, services[id], cfg)
		if err != nil {
			return err
		}
		for _, dep := range deps {
			if err := dfs(dep, append(path, id)); err != nil {
				return err
			}
		}
		stack[id] = false
		return nil
	}

	for id := range services {
		if err := dfs(id, nil); err != nil {
			return err
		}
	}
	return nil
}

func computeGetterErrors(services map[string]*serviceDef, cfg *di.Config, decoratorsByBase map[string][]*serviceDef) (map[string]bool, map[string]bool) {
	buildCan := map[string]bool{}
	getterCan := map[string]bool{}
	for id, svc := range services {
		getterCan[id] = svc.constructor.returnsError
		buildCan[id] = svc.constructor.returnsError
	}

	changed := true
	for changed {
		changed = false
		for id, svc := range services {
			can := svc.constructor.returnsError
			deps, _ := buildDeps(id, svc, cfg)
			for _, dep := range deps {
				if getterCan[dep] {
					can = true
				}
			}
			if buildCan[id] != can {
				buildCan[id] = can
				changed = true
			}
		}

		for id, svc := range services {
			newVal := buildCan[id]
			if svc.isDecorator {
				base := svc.decorates
				newVal = false
				if buildCan[base] {
					newVal = true
				}
				decs := decoratorsByBase[base]
				for _, d := range decs {
					if buildCan[d.id] {
						newVal = true
					}
					if d.id == id {
						break
					}
				}
			} else if decs := decoratorsByBase[id]; len(decs) > 0 {
				newVal = buildCan[id]
				for _, d := range decs {
					if buildCan[d.id] {
						newVal = true
					}
				}
			}
			if getterCan[id] != newVal {
				getterCan[id] = newVal
				changed = true
			}
		}
	}

	return buildCan, getterCan
}

func buildDeps(id string, svc *serviceDef, cfg *di.Config) ([]string, error) {
	deps := []string{}
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
		if _, ok := tt.Underlying().(*types.Interface); ok {
			return fmt.Errorf("service type must not be interface")
		}
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
	default:
		return nil, fmt.Errorf("unsupported literal type %q", node.Tag)
	}
}

func getterType(svc *serviceDef, services map[string]*serviceDef, decoratorsByBase map[string][]*serviceDef) types.Type {
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
