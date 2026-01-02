package ir

import (
	"fmt"
	"go/types"
	"sort"
)

// decoratorExpander rewrites decorator syntax into plain services and aliases.
type decoratorExpander struct{}

// expand converts decorator chains into standard service references.
func (e *decoratorExpander) expand(ctx *buildContext) error {
	if err := e.detectDecoratorCycles(ctx.services); err != nil {
		return err
	}

	usedIDs := make(map[string]bool, len(ctx.services))
	for id := range ctx.services {
		usedIDs[id] = true
	}

	rawBaseByResolved := make(map[string]*Service)

	for _, baseID := range ctx.order {
		base := ctx.services[baseID]
		if base == nil || len(base.Decorators) == 0 {
			continue
		}
		if base.Decorates != nil {
			return fmt.Errorf("decorator %q cannot be decorated", base.ID)
		}

		decorators := append([]*Service(nil), base.Decorators...)
		sort.Slice(decorators, func(i, j int) bool {
			return decorators[i].Priority < decorators[j].Priority
		})

		resolvedBase := e.resolveAliasTarget(base)
		rawBase := rawBaseByResolved[resolvedBase.ID]
		if rawBase == nil {
			rawBaseID := uniqueInternalID(usedIDs, "base", resolvedBase.ID)
			rawBase = cloneService(resolvedBase, rawBaseID)
			rawBase.Public = false
			rawBase.Shared = false
			rawBase.Tags = nil
			rawBase.Alias = nil
			rawBase.Decorates = nil
			rawBase.Decorators = nil
			ctx.services[rawBaseID] = rawBase
			rawBaseByResolved[resolvedBase.ID] = rawBase
		}

		prev := rawBase
		for i, dec := range decorators {
			isOutermost := i == len(decorators)-1
			if isOutermost {
				base.Constructor = cloneConstructor(dec.Constructor)
				base.Type = dec.Type
				base.Decorates = nil
				base.Decorators = nil
				base.Alias = nil
				e.rewriteInnerArgs(base.Constructor, prev)
				break
			}

			internalID := uniqueInternalID(usedIDs, "decorator", dec.ID)
			internal := cloneService(dec, internalID)
			internal.Public = false
			internal.Shared = false
			internal.Tags = nil
			internal.Alias = nil
			internal.Decorates = nil
			internal.Decorators = nil
			e.rewriteInnerArgs(internal.Constructor, prev)

			ctx.services[internalID] = internal
			prev = internal
		}

		chainShared := base.Shared
		for _, dec := range decorators {
			if dec.Shared {
				chainShared = true
				break
			}
		}
		base.Shared = chainShared

		for _, dec := range decorators {
			dec.Alias = base
			dec.Constructor = nil
			dec.Type = base.Type
			dec.Decorates = nil
			dec.Decorators = nil
		}
	}

	e.validateInnerArgs(ctx.services)
	e.rebuildOrder(ctx)
	return nil
}

func (e *decoratorExpander) validateInnerArgs(services map[string]*Service) error {
	for _, svc := range services {
		if svc.Constructor == nil {
			continue
		}
		for _, arg := range svc.Constructor.Args {
			if arg.Kind == InnerArg {
				return fmt.Errorf("@.inner used outside decorator")
			}
		}
	}
	return nil
}

func (e *decoratorExpander) rewriteInnerArgs(cons *Constructor, innerSvc *Service) {
	if cons == nil || innerSvc == nil {
		return
	}
	for _, arg := range cons.Args {
		if arg.Kind != InnerArg {
			continue
		}
		arg.Kind = ServiceRefArg
		arg.Service = innerSvc
		arg.Inner = false
	}
}

func (e *decoratorExpander) resolveAliasTarget(svc *Service) *Service {
	for svc != nil && svc.Alias != nil {
		svc = svc.Alias
	}
	return svc
}

func (e *decoratorExpander) detectDecoratorCycles(services map[string]*Service) error {
	decorators := make(map[string]*Service)
	for id, svc := range services {
		if svc.Decorates != nil {
			decorators[id] = svc
		}
	}

	visited := make(map[string]bool)
	stack := make(map[string]bool)
	var dfs func(*Service, []string) error
	dfs = func(svc *Service, path []string) error {
		if svc == nil {
			return nil
		}
		if stack[svc.ID] {
			cycle := append(path, svc.ID)
			return fmt.Errorf("circular decorator chain: %s", joinIDs(cycle))
		}
		if visited[svc.ID] {
			return nil
		}
		visited[svc.ID] = true
		stack[svc.ID] = true

		if svc.Decorates != nil {
			if err := dfs(svc.Decorates, append(path, svc.ID)); err != nil {
				return err
			}
		}

		stack[svc.ID] = false
		return nil
	}

	for _, svc := range decorators {
		if err := dfs(svc, nil); err != nil {
			return err
		}
	}
	return nil
}

func (e *decoratorExpander) rebuildOrder(ctx *buildContext) {
	ctx.order = ctx.order[:0]
	for id := range ctx.services {
		ctx.order = append(ctx.order, id)
	}
	sort.Strings(ctx.order)
}

func uniqueInternalID(used map[string]bool, prefix, id string) string {
	base := fmt.Sprintf("__decorator_%s__%s", prefix, id)
	if !used[base] {
		used[base] = true
		return base
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s_%d", base, i)
		if !used[candidate] {
			used[candidate] = true
			return candidate
		}
	}
}

func cloneService(src *Service, id string) *Service {
	if src == nil {
		return nil
	}
	clone := *src
	clone.ID = id
	clone.Tags = nil
	clone.Decorates = nil
	clone.Decorators = nil
	clone.Alias = nil
	clone.Dependencies = nil
	clone.CanError = false
	clone.BuildCanError = false
	clone.Constructor = cloneConstructor(src.Constructor)
	return &clone
}

func cloneConstructor(src *Constructor) *Constructor {
	if src == nil {
		return nil
	}
	clone := *src
	if len(src.Args) > 0 {
		clone.Args = make([]*Argument, len(src.Args))
		for i, arg := range src.Args {
			if arg == nil {
				continue
			}
			argClone := *arg
			clone.Args[i] = &argClone
		}
	}
	if len(src.Params) > 0 {
		clone.Params = append([]types.Type(nil), src.Params...)
	}
	if len(src.TypeArgs) > 0 {
		clone.TypeArgs = append([]types.Type(nil), src.TypeArgs...)
	}
	return &clone
}

func joinIDs(ids []string) string {
	switch len(ids) {
	case 0:
		return ""
	case 1:
		return ids[0]
	}
	out := ids[0]
	for _, id := range ids[1:] {
		out += " -> " + id
	}
	return out
}
