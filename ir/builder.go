package ir

import (
	"errors"
	"fmt"
	"go/types"
	"sort"
	"strings"

	di "github.com/asp24/gendi"
)

// TypeResolver resolves type strings to Go types.
type TypeResolver interface {
	LookupType(typeStr string) (types.Type, error)
	LookupFunc(pkgPath, name string) (*types.Func, error)
	LookupMethod(recv types.Type, name string) (*types.Func, error)
}

// Builder constructs an IR Container from raw config.
type Builder struct {
	cfg      *di.Config
	resolver TypeResolver

	// Intermediate state
	services   map[string]*Service
	parameters map[string]*Parameter
	tags       map[string]*Tag
	order      []string
}

// NewBuilder creates a new IR builder.
func NewBuilder(cfg *di.Config, resolver TypeResolver) *Builder {
	return &Builder{
		cfg:        cfg,
		resolver:   resolver,
		services:   make(map[string]*Service),
		parameters: make(map[string]*Parameter),
		tags:       make(map[string]*Tag),
	}
}

// Build constructs the IR Container.
func (b *Builder) Build() (*Container, error) {
	if b.cfg.Services == nil || len(b.cfg.Services) == 0 {
		return nil, errors.New("no services defined")
	}

	if err := b.buildParameters(); err != nil {
		return nil, err
	}
	if err := b.buildTags(); err != nil {
		return nil, err
	}
	if err := b.buildServices(); err != nil {
		return nil, err
	}
	if err := b.resolveConstructors(); err != nil {
		return nil, err
	}
	if err := b.resolveDecorators(); err != nil {
		return nil, err
	}
	if err := b.resolveTaggedServices(); err != nil {
		return nil, err
	}
	if err := b.resolveDependencies(); err != nil {
		return nil, err
	}
	if err := b.validatePublicServices(); err != nil {
		return nil, err
	}
	if err := b.detectCycles(); err != nil {
		return nil, err
	}
	b.computeErrorPropagation()

	return b.buildContainer(), nil
}

func (b *Builder) buildParameters() error {
	for name, param := range b.cfg.Parameters {
		if param.Type == "" {
			return fmt.Errorf("parameter %q missing type", name)
		}
		paramType, err := b.resolver.LookupType(param.Type)
		if err != nil {
			return fmt.Errorf("parameter %q type: %w", name, err)
		}

		litVal, err := b.convertLiteral(param.Value, paramType)
		if err != nil {
			return fmt.Errorf("parameter %q value: %w", name, err)
		}

		b.parameters[name] = &Parameter{
			Name:  name,
			Type:  paramType,
			Value: litVal,
		}
	}
	return nil
}

func (b *Builder) buildTags() error {
	for name, tag := range b.cfg.Tags {
		if tag.ElementType == "" {
			return fmt.Errorf("tag %q missing element_type", name)
		}
		elemType, err := b.resolver.LookupType(tag.ElementType)
		if err != nil {
			return fmt.Errorf("tag %q element_type: %w", name, err)
		}
		b.tags[name] = &Tag{
			Name:        name,
			ElementType: elemType,
			SortBy:      tag.SortBy,
			Services:    []*Service{},
		}
	}
	return nil
}

func (b *Builder) buildServices() error {
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

		irSvc := &Service{
			ID:       id,
			Shared:   shared,
			Public:   svc.Public,
			Priority: svc.DecorationPriority,
			Tags:     []*ServiceTag{},
		}

		// Build service tags
		for _, st := range svc.Tags {
			tag, ok := b.tags[st.Name]
			if !ok {
				return fmt.Errorf("service %q references unknown tag %q", id, st.Name)
			}
			irSvc.Tags = append(irSvc.Tags, &ServiceTag{
				Tag:        tag,
				Attributes: st.Attributes,
			})
		}

		b.services[id] = irSvc
	}
	sort.Strings(b.order)
	return nil
}

func (b *Builder) resolveConstructors() error {
	tracker := &resolutionTracker{
		resolving: make(map[string]bool),
		resolved:  make(map[string]bool),
	}

	var resolve func(id string) error
	resolve = func(id string) error {
		if tracker.resolved[id] {
			return nil
		}
		if tracker.resolving[id] {
			return fmt.Errorf("circular constructor reference at %q", id)
		}
		tracker.resolving[id] = true
		defer func() {
			tracker.resolving[id] = false
			tracker.resolved[id] = true
		}()

		svc := b.services[id]
		cfg := b.cfg.Services[id]

		if cfg.Alias != "" {
			return b.resolveAlias(svc, cfg, resolve)
		}
		return b.resolveConstructor(svc, cfg, resolve)
	}

	for _, id := range b.order {
		if err := resolve(id); err != nil {
			return err
		}
	}
	return nil
}

func (b *Builder) resolveAlias(svc *Service, cfg *di.Service, resolve func(string) error) error {
	if cfg.Constructor.Func != "" || cfg.Constructor.Method != "" || len(cfg.Constructor.Args) > 0 {
		return fmt.Errorf("service %q alias cannot define constructor", svc.ID)
	}
	if cfg.Decorates != "" {
		return fmt.Errorf("service %q alias cannot be a decorator", svc.ID)
	}

	if err := resolve(cfg.Alias); err != nil {
		return err
	}

	target, ok := b.services[cfg.Alias]
	if !ok {
		return fmt.Errorf("service %q alias target %q not found", svc.ID, cfg.Alias)
	}

	svc.Alias = target
	svc.Type = target.Type

	if cfg.Type != "" {
		declType, err := b.resolver.LookupType(cfg.Type)
		if err != nil {
			return fmt.Errorf("service %q type: %w", svc.ID, err)
		}
		if !types.AssignableTo(svc.Type, declType) {
			return fmt.Errorf("service %q type mismatch", svc.ID)
		}
	}
	return nil
}

func (b *Builder) resolveConstructor(svc *Service, cfg *di.Service, resolve func(string) error) error {
	cons := cfg.Constructor
	if cons.Func == "" && cons.Method == "" {
		return fmt.Errorf("service %q missing constructor", svc.ID)
	}
	if cons.Func != "" && cons.Method != "" {
		return fmt.Errorf("service %q has both func and method constructors", svc.ID)
	}

	var irCons *Constructor
	var err error

	if cons.Func != "" {
		irCons, err = b.resolveFuncConstructor(svc.ID, cons)
	} else {
		irCons, err = b.resolveMethodConstructor(svc.ID, cons, resolve)
	}
	if err != nil {
		return err
	}

	// Resolve arguments
	if len(cons.Args) != len(irCons.Params) {
		return fmt.Errorf("service %q constructor args count mismatch: expected %d got %d",
			svc.ID, len(irCons.Params), len(cons.Args))
	}

	irCons.Args = make([]*Argument, len(cons.Args))
	for i, arg := range cons.Args {
		irArg, err := b.resolveArgument(svc.ID, i, arg, irCons.Params[i])
		if err != nil {
			return err
		}
		irCons.Args[i] = irArg
	}

	svc.Constructor = irCons
	svc.Type = irCons.ResultType

	if cfg.Type != "" {
		declType, err := b.resolver.LookupType(cfg.Type)
		if err != nil {
			return fmt.Errorf("service %q type: %w", svc.ID, err)
		}
		if !types.AssignableTo(svc.Type, declType) {
			return fmt.Errorf("service %q type mismatch", svc.ID)
		}
	}

	return nil
}

func (b *Builder) resolveFuncConstructor(id string, cons di.Constructor) (*Constructor, error) {
	pkgPath, name, err := splitPkgSymbol(cons.Func)
	if err != nil {
		return nil, fmt.Errorf("service %q constructor.func: %w", id, err)
	}

	fn, err := b.resolver.LookupFunc(pkgPath, name)
	if err != nil {
		return nil, fmt.Errorf("service %q constructor.func: %w", id, err)
	}

	sig := fn.Type().(*types.Signature)
	resultType, returnsErr, err := validateConstructorSignature(sig)
	if err != nil {
		return nil, fmt.Errorf("service %q constructor.func: %w", id, err)
	}

	return &Constructor{
		Kind:         FuncConstructor,
		Func:         fn,
		Params:       signatureParams(sig),
		ResultType:   resultType,
		ReturnsError: returnsErr,
	}, nil
}

func (b *Builder) resolveMethodConstructor(id string, cons di.Constructor, resolve func(string) error) (*Constructor, error) {
	methodRef := cons.Method
	if !strings.HasPrefix(methodRef, "@") {
		return nil, fmt.Errorf("service %q constructor.method must start with @", id)
	}

	methodRef = methodRef[1:]
	parts := strings.Split(methodRef, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("service %q constructor.method invalid format", id)
	}

	methodName := parts[len(parts)-1]
	recvID := strings.Join(parts[:len(parts)-1], ".")
	if recvID == "" || methodName == "" {
		return nil, fmt.Errorf("service %q constructor.method invalid format", id)
	}

	if err := resolve(recvID); err != nil {
		return nil, err
	}

	recvSvc, ok := b.services[recvID]
	if !ok {
		return nil, fmt.Errorf("service %q constructor.method unknown receiver service %q", id, recvID)
	}

	meth, err := b.resolver.LookupMethod(recvSvc.Type, methodName)
	if err != nil {
		return nil, fmt.Errorf("service %q constructor.method: %w", id, err)
	}

	sig := meth.Type().(*types.Signature)
	resultType, returnsErr, err := validateConstructorSignature(sig)
	if err != nil {
		return nil, fmt.Errorf("service %q constructor.method: %w", id, err)
	}

	return &Constructor{
		Kind:         MethodConstructor,
		Func:         meth,
		Receiver:     recvSvc,
		Params:       signatureParams(sig),
		ResultType:   resultType,
		ReturnsError: returnsErr,
	}, nil
}

func (b *Builder) resolveArgument(svcID string, idx int, arg di.Argument, paramType types.Type) (*Argument, error) {
	irArg := &Argument{Type: paramType}

	switch arg.Kind {
	case di.ArgServiceRef:
		dep, ok := b.services[arg.Value]
		if !ok {
			return nil, fmt.Errorf("service %q arg[%d]: unknown service %q", svcID, idx, arg.Value)
		}
		irArg.Kind = ServiceRefArg
		irArg.Service = dep

	case di.ArgInner:
		irArg.Kind = InnerArg
		irArg.Inner = true

	case di.ArgParam:
		param, ok := b.parameters[arg.Value]
		if !ok {
			// Parameter might be provided at runtime
			irArg.Kind = ParamRefArg
			irArg.Parameter = &Parameter{Name: arg.Value, Type: paramType}
		} else {
			if !types.AssignableTo(param.Type, paramType) {
				return nil, fmt.Errorf("service %q arg[%d]: parameter %q type mismatch", svcID, idx, arg.Value)
			}
			irArg.Kind = ParamRefArg
			irArg.Parameter = param
		}

	case di.ArgTagged:
		tag, ok := b.tags[arg.Value]
		if !ok {
			return nil, fmt.Errorf("service %q arg[%d]: unknown tag %q", svcID, idx, arg.Value)
		}
		irArg.Kind = TaggedArg
		irArg.Tag = tag

	default: // Literal
		litVal, err := b.convertLiteral(arg.Literal, paramType)
		if err != nil {
			return nil, fmt.Errorf("service %q arg[%d]: %w", svcID, idx, err)
		}
		irArg.Kind = LiteralArg
		irArg.Literal = litVal
	}

	return irArg, nil
}

func (b *Builder) resolveDecorators() error {
	for _, svc := range b.services {
		cfg := b.cfg.Services[svc.ID]
		if cfg.Decorates == "" {
			continue
		}

		base, ok := b.services[cfg.Decorates]
		if !ok {
			return fmt.Errorf("decorator %q decorates unknown service %q", svc.ID, cfg.Decorates)
		}

		svc.Decorates = base
		base.Decorators = append(base.Decorators, svc)
	}

	// Sort decorators by priority
	for _, svc := range b.services {
		if len(svc.Decorators) > 1 {
			sort.Slice(svc.Decorators, func(i, j int) bool {
				return svc.Decorators[i].Priority < svc.Decorators[j].Priority
			})
		}
	}

	return nil
}

func (b *Builder) resolveTaggedServices() error {
	for _, svc := range b.services {
		for _, st := range svc.Tags {
			st.Tag.Services = append(st.Tag.Services, svc)
		}
	}
	// Could add sorting by attributes here if SortBy is specified
	return nil
}

func (b *Builder) resolveDependencies() error {
	for _, svc := range b.services {
		if svc.IsAlias() {
			svc.Dependencies = []*Service{svc.Alias}
			continue
		}
		if svc.Constructor == nil {
			continue
		}

		deps := make(map[string]*Service)

		// Method receiver is a dependency
		if svc.Constructor.Kind == MethodConstructor && svc.Constructor.Receiver != nil {
			deps[svc.Constructor.Receiver.ID] = svc.Constructor.Receiver
		}

		for _, arg := range svc.Constructor.Args {
			switch arg.Kind {
			case ServiceRefArg:
				if arg.Service != nil {
					deps[arg.Service.ID] = arg.Service
				}
			case InnerArg:
				if svc.Decorates != nil {
					deps[svc.Decorates.ID] = svc.Decorates
				}
			case TaggedArg:
				if arg.Tag != nil {
					for _, tagged := range arg.Tag.Services {
						deps[tagged.ID] = tagged
					}
				}
			}
		}

		svc.Dependencies = make([]*Service, 0, len(deps))
		for _, dep := range deps {
			svc.Dependencies = append(svc.Dependencies, dep)
		}
	}
	return nil
}

func (b *Builder) validatePublicServices() error {
	for _, svc := range b.services {
		if svc.Public {
			return nil
		}
	}
	return errors.New("at least one public service is required")
}

func (b *Builder) detectCycles() error {
	visited := make(map[string]bool)
	stack := make(map[string]bool)

	var dfs func(svc *Service, path []string) error
	dfs = func(svc *Service, path []string) error {
		if stack[svc.ID] {
			cycle := append(path, svc.ID)
			return fmt.Errorf("circular dependency: %s", strings.Join(cycle, " -> "))
		}
		if visited[svc.ID] {
			return nil
		}
		visited[svc.ID] = true
		stack[svc.ID] = true

		for _, dep := range svc.Dependencies {
			if err := dfs(dep, append(path, svc.ID)); err != nil {
				return err
			}
		}

		stack[svc.ID] = false
		return nil
	}

	for _, svc := range b.services {
		if err := dfs(svc, nil); err != nil {
			return err
		}
	}
	return nil
}

func (b *Builder) computeErrorPropagation() {
	// Initialize from constructors
	for _, svc := range b.services {
		if svc.Constructor != nil {
			svc.BuildCanError = svc.Constructor.ReturnsError
		}
	}

	// Propagate until stable
	changed := true
	for changed {
		changed = false

		// Build errors propagate from dependencies
		for _, svc := range b.services {
			if svc.BuildCanError {
				continue
			}
			for _, dep := range svc.Dependencies {
				if dep.CanError {
					svc.BuildCanError = true
					changed = true
					break
				}
			}
		}

		// Getter errors include decorator errors
		for _, svc := range b.services {
			newVal := svc.BuildCanError
			if len(svc.Decorators) > 0 {
				for _, dec := range svc.Decorators {
					if dec.BuildCanError {
						newVal = true
						break
					}
				}
			}
			if svc.CanError != newVal {
				svc.CanError = newVal
				changed = true
			}
		}
	}
}

func (b *Builder) buildContainer() *Container {
	return &Container{
		Services:     b.services,
		Parameters:   b.parameters,
		Tags:         b.tags,
		ServiceOrder: b.order,
	}
}

func (b *Builder) convertLiteral(lit di.Literal, targetType types.Type) (LiteralValue, error) {
	if isDuration(targetType) {
		return b.convertDurationLiteral(lit)
	}

	switch lit.Kind {
	case di.LiteralString:
		return LiteralValue{Type: StringLiteral, Value: lit.String()}, nil
	case di.LiteralInt:
		return LiteralValue{Type: IntLiteral, Value: lit.Int()}, nil
	case di.LiteralFloat:
		return LiteralValue{Type: FloatLiteral, Value: lit.Float()}, nil
	case di.LiteralBool:
		return LiteralValue{Type: BoolLiteral, Value: lit.Bool()}, nil
	case di.LiteralNull:
		return LiteralValue{Type: NullLiteral, Value: nil}, nil
	default:
		return LiteralValue{}, fmt.Errorf("unsupported literal kind %d", lit.Kind)
	}
}

func (b *Builder) convertDurationLiteral(lit di.Literal) (LiteralValue, error) {
	// Could be string "1s" or integer nanoseconds
	switch lit.Kind {
	case di.LiteralString:
		// Parse as duration string - will be handled by generator
		return LiteralValue{Type: DurationLiteral, Value: lit.String()}, nil
	case di.LiteralInt:
		return LiteralValue{Type: DurationLiteral, Value: lit.Int()}, nil
	default:
		return LiteralValue{}, fmt.Errorf("duration must be string or int")
	}
}

type resolutionTracker struct {
	resolving map[string]bool
	resolved  map[string]bool
}

func splitPkgSymbol(s string) (string, string, error) {
	idx := strings.LastIndex(s, ".")
	if idx <= 0 || idx == len(s)-1 {
		return "", "", fmt.Errorf("invalid qualified name %q", s)
	}
	return s[:idx], s[idx+1:], nil
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
		if !types.Identical(errType, types.Universe.Lookup("error").Type()) {
			return nil, false, fmt.Errorf("second return value must be error")
		}
		returnsErr = true
	}
	return resType, returnsErr, nil
}

func signatureParams(sig *types.Signature) []types.Type {
	params := make([]types.Type, sig.Params().Len())
	for i := 0; i < sig.Params().Len(); i++ {
		params[i] = sig.Params().At(i).Type()
	}
	return params
}
