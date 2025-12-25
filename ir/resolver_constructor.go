package ir

import (
	"fmt"
	"go/types"
	"strings"

	di "github.com/asp24/gendi"
)

// constructorResolver resolves service constructors (functions, methods, and aliases)
type constructorResolver struct{}

// resolve resolves all service constructors with circular reference detection
func (r *constructorResolver) resolve(ctx *buildContext) error {
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

		svc := ctx.services[id]
		cfg := ctx.cfg.Services[id]

		if cfg.Alias != "" {
			return r.resolveAlias(ctx, svc, cfg, resolveService)
		}
		return r.resolveConstructor(ctx, svc, cfg, resolveService)
	}

	for _, id := range ctx.order {
		if err := resolveService(id); err != nil {
			return err
		}
	}
	return nil
}

// resolveAlias resolves an alias service
func (r *constructorResolver) resolveAlias(ctx *buildContext, svc *Service, cfg *di.Service, resolve func(string) error) error {
	if cfg.Constructor.Func != "" || cfg.Constructor.Method != "" || len(cfg.Constructor.Args) > 0 {
		return fmt.Errorf("service %q alias cannot define constructor", svc.ID)
	}
	if cfg.Decorates != "" {
		return fmt.Errorf("service %q alias cannot be a decorator", svc.ID)
	}

	if err := resolve(cfg.Alias); err != nil {
		return err
	}

	target, ok := ctx.services[cfg.Alias]
	if !ok {
		return fmt.Errorf("service %q alias target %q not found", svc.ID, cfg.Alias)
	}

	svc.Alias = target
	svc.Type = target.Type

	if cfg.Type != "" {
		declType, err := ctx.resolver.LookupType(cfg.Type)
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
func (r *constructorResolver) resolveConstructor(ctx *buildContext, svc *Service, cfg *di.Service, resolve func(string) error) error {
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
		irCons, err = r.resolveFuncConstructor(ctx, svc.ID, cons)
	} else {
		irCons, err = r.resolveMethodConstructor(ctx, svc.ID, cons, resolve)
	}
	if err != nil {
		return err
	}

	// Resolve arguments
	if len(cons.Args) != len(irCons.Params) {
		return fmt.Errorf("service %q constructor args count mismatch: expected %d got %d",
			svc.ID, len(irCons.Params), len(cons.Args))
	}

	argResolver := &argumentResolver{}
	irCons.Args = make([]*Argument, len(cons.Args))
	for i, arg := range cons.Args {
		irArg, err := argResolver.resolve(ctx, svc.ID, i, arg, irCons.Params[i])
		if err != nil {
			return err
		}
		irCons.Args[i] = irArg
	}

	svc.Constructor = irCons
	svc.Type = irCons.ResultType

	if cfg.Type != "" {
		declType, err := ctx.resolver.LookupType(cfg.Type)
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
func (r *constructorResolver) resolveFuncConstructor(ctx *buildContext, id string, cons di.Constructor) (*Constructor, error) {
	pkgPath, name, err := splitPkgSymbol(cons.Func)
	if err != nil {
		return nil, fmt.Errorf("service %q constructor.func: %w", id, err)
	}

	fn, err := ctx.resolver.LookupFunc(pkgPath, name)
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

// resolveMethodConstructor resolves a method constructor
func (r *constructorResolver) resolveMethodConstructor(ctx *buildContext, id string, cons di.Constructor, resolve func(string) error) (*Constructor, error) {
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

	recvSvc, ok := ctx.services[recvID]
	if !ok {
		return nil, fmt.Errorf("service %q constructor.method unknown receiver service %q", id, recvID)
	}

	meth, err := ctx.resolver.LookupMethod(recvSvc.Type, methodName)
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
