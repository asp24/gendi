package generator

import (
	"fmt"
	"go/types"
)

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
func resolveAliasService(svc *serviceDef, services map[string]*serviceDef, loader *TypeLoader, resolveFunc func(string) error) error {
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
func resolveConstructorService(svc *serviceDef, services map[string]*serviceDef, loader *TypeLoader, resolveFunc func(string) error) error {
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
