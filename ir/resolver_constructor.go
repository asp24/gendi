package ir

import (
	"fmt"
	"go/types"
	"strings"

	di "github.com/asp24/gendi"
	"github.com/asp24/gendi/internal/typeutil"
)

// constructorResolver resolves service constructors (functions, methods, and aliases)
type constructorResolver struct {
	resolver TypeResolver
}

// resolve resolves all service constructors with circular reference detection
func (r *constructorResolver) resolve(cfg *di.Config, container *Container) error {
	tracker := newResolutionTracker()

	var resolveService func(id string) error
	resolveService = func(id string) error {
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

		svc := container.Services[id]
		cfgService := cfg.Services[id]

		if cfgService.Alias != "" {
			return r.resolveAlias(container, svc, &cfgService, resolveService)
		}
		return r.resolveConstructor(container, svc, &cfgService, resolveService)
	}

	for _, id := range container.ServiceIDsPostOrder() {
		if err := resolveService(id); err != nil {
			return err
		}
	}
	return nil
}

// resolveAlias resolves an alias service
func (r *constructorResolver) resolveAlias(container *Container, svc *Service, cfg *di.Service, resolve func(string) error) error {
	if cfg.Constructor.Func != "" || cfg.Constructor.Method != "" || len(cfg.Constructor.Args) > 0 {
		return fmt.Errorf("service %q alias cannot define constructor", svc.ID)
	}
	if cfg.Decorates != "" {
		return fmt.Errorf("service %q alias cannot be a decorator", svc.ID)
	}

	if err := resolve(cfg.Alias); err != nil {
		return err
	}

	target, ok := container.Services[cfg.Alias]
	if !ok {
		return fmt.Errorf("service %q alias target %q not found", svc.ID, cfg.Alias)
	}

	svc.Alias = target
	svc.Type = target.Type

	if cfg.Type != "" {
		declType, err := r.resolver.LookupType(cfg.Type)
		if err != nil {
			return fmt.Errorf("service %q type: %w", svc.ID, err)
		}
		if !types.AssignableTo(svc.Type, declType) {
			return fmt.Errorf("service %q type mismatch", svc.ID)
		}
	}
	return nil
}

// resolveConstructor resolves a function or method constructor
func (r *constructorResolver) resolveConstructor(container *Container, svc *Service, cfg *di.Service, resolve func(string) error) error {
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
		irCons, err = r.resolveFuncConstructor(svc.ID, cons)
	} else {
		irCons, err = r.resolveMethodConstructor(container, svc.ID, cons, resolve)
	}
	if err != nil {
		return err
	}

	// Resolve arguments
	expectedMin := len(irCons.Params)
	if irCons.Variadic {
		expectedMin-- // Variadic parameter can accept 0 or more args
	}

	if !irCons.Variadic && len(cons.Args) != len(irCons.Params) {
		return fmt.Errorf("service %q constructor args count mismatch: expected %d got %d",
			svc.ID, len(irCons.Params), len(cons.Args))
	}

	if irCons.Variadic && len(cons.Args) < expectedMin {
		return fmt.Errorf("service %q constructor requires at least %d args, got %d",
			svc.ID, expectedMin, len(cons.Args))
	}

	argResolver := &argumentResolver{}
	irCons.Args = make([]*Argument, len(cons.Args))
	for i, arg := range cons.Args {
		// For variadic functions, all args after the last non-variadic param
		// use the variadic param's element type
		paramIdx := i
		if irCons.Variadic && i >= len(irCons.Params)-1 {
			paramIdx = len(irCons.Params) - 1
		}

		paramType := irCons.Params[paramIdx]
		// If this is a variadic parameter, get the slice element type
		if irCons.Variadic && paramIdx == len(irCons.Params)-1 {
			if sliceType, ok := paramType.(*types.Slice); ok {
				paramType = sliceType.Elem()
			}
		}

		irArg, err := argResolver.resolve(container, svc.ID, i, arg, paramType)
		if err != nil {
			return err
		}
		irCons.Args[i] = irArg
	}

	svc.Constructor = irCons
	svc.Type = irCons.ResultType

	if cfg.Type != "" {
		declType, err := r.resolver.LookupType(cfg.Type)
		if err != nil {
			return fmt.Errorf("service %q type: %w", svc.ID, err)
		}
		if !types.AssignableTo(svc.Type, declType) {
			return fmt.Errorf("service %q type mismatch", svc.ID)
		}
	}

	return nil
}

// resolveFuncConstructor resolves a function constructor
func (r *constructorResolver) resolveFuncConstructor(id string, cons di.Constructor) (*Constructor, error) {
	pkgPath, name, typeParamStrs, err := typeutil.SplitQualifiedNameWithTypeParams(cons.Func)
	if err != nil {
		return nil, fmt.Errorf("service %q constructor.func: %w", id, err)
	}

	fn, err := r.resolver.LookupFunc(pkgPath, name)
	if err != nil {
		return nil, fmt.Errorf("service %q constructor.func: %w", id, err)
	}

	var sig *types.Signature
	var typeArgs []types.Type

	rawSig := fn.Type().(*types.Signature)

	if len(typeParamStrs) > 0 {
		// Generic function - instantiate with type arguments
		sig, typeArgs, err = r.resolver.InstantiateFunc(fn, typeParamStrs)
		if err != nil {
			return nil, fmt.Errorf("service %q constructor.func: %w", id, err)
		}
	} else {
		// Check if function is generic but no type arguments provided
		if tp := rawSig.TypeParams(); tp != nil && tp.Len() > 0 {
			return nil, fmt.Errorf("service %q constructor.func: generic function %s requires %d type argument(s)",
				id, name, tp.Len())
		}
		sig = rawSig
	}

	resultType, returnsErr, err := validateConstructorSignature(sig)
	if err != nil {
		return nil, fmt.Errorf("service %q constructor.func: %w", id, err)
	}

	return &Constructor{
		Kind:         FuncConstructor,
		Func:         fn,
		TypeArgs:     typeArgs,
		Params:       signatureParams(sig),
		ResultType:   resultType,
		ReturnsError: returnsErr,
		Variadic:     sig.Variadic(),
	}, nil
}

// resolveMethodConstructor resolves a method constructor
func (r *constructorResolver) resolveMethodConstructor(container *Container, id string, cons di.Constructor, resolve func(string) error) (*Constructor, error) {
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

	recvSvc, ok := container.Services[recvID]
	if !ok {
		return nil, fmt.Errorf("service %q constructor.method unknown receiver service %q", id, recvID)
	}

	meth, err := r.resolver.LookupMethod(recvSvc.Type, methodName)
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
		Variadic:     sig.Variadic(),
	}, nil
}
