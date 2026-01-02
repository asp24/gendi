package ir

import (
	"fmt"
	"sort"
)

// decoratorResolver links decorators to their base services
type decoratorResolver struct{}

type decoratorResolverState struct {
	decoratesByID    map[string]string
	decoratorsByBase map[string][]*Service
	priorityByID     map[string]int
}

// resolve links decorators and expands them into plain services and aliases.
func (r *decoratorResolver) resolve(ctx *buildContext) error {
	state, err := r.buildState(ctx)
	if err != nil {
		return err
	}
	return r.expandDecorators(ctx, state)
}

func (r *decoratorResolver) buildState(ctx *buildContext) (*decoratorResolverState, error) {
	state := &decoratorResolverState{
		decoratesByID:    make(map[string]string),
		decoratorsByBase: make(map[string][]*Service),
		priorityByID:     make(map[string]int),
	}

	for _, svc := range ctx.services {
		cfg := ctx.cfg.Services[svc.ID]
		if cfg.Decorates == "" {
			continue
		}

		base, ok := ctx.services[cfg.Decorates]
		if !ok {
			return nil, fmt.Errorf("decorator %q decorates unknown service %q", svc.ID, cfg.Decorates)
		}

		state.decoratesByID[svc.ID] = base.ID
		state.decoratorsByBase[base.ID] = append(state.decoratorsByBase[base.ID], svc)
		state.priorityByID[svc.ID] = cfg.DecorationPriority
	}

	for baseID, decs := range state.decoratorsByBase {
		if len(decs) > 1 {
			sort.Slice(decs, func(i, j int) bool {
				return state.priorityByID[decs[i].ID] < state.priorityByID[decs[j].ID]
			})
			state.decoratorsByBase[baseID] = decs
		}
	}

	return state, nil
}

// expandDecorators rewrites decorator syntax into plain services and aliases.
func (r *decoratorResolver) expandDecorators(ctx *buildContext, state *decoratorResolverState) error {
	if err := r.detectDecoratorCycles(state.decoratesByID); err != nil {
		return err
	}

	usedIDs := make(map[string]bool, len(ctx.services))
	for id := range ctx.services {
		usedIDs[id] = true
	}

	rawBaseByResolved := make(map[string]*Service)

	for _, baseID := range ctx.order {
		base := ctx.services[baseID]
		decorators := state.decoratorsByBase[baseID]
		if base == nil || len(decorators) == 0 {
			continue
		}
		if _, ok := state.decoratesByID[base.ID]; ok {
			return fmt.Errorf("decorator %q cannot be decorated", base.ID)
		}
		decorators = append([]*Service(nil), decorators...)

		resolvedBase := r.resolveAliasTarget(base)
		rawBase := rawBaseByResolved[resolvedBase.ID]
		if rawBase == nil {
			rawBaseID := uniqueInternalID(usedIDs, "base", resolvedBase.ID)
			rawBase = cloneService(resolvedBase, rawBaseID)
			rawBase.Public = false
			rawBase.Shared = false
			rawBase.Tags = nil
			rawBase.Alias = nil
			ctx.services[rawBaseID] = rawBase
			rawBaseByResolved[resolvedBase.ID] = rawBase
		}

		prev := rawBase
		for i, dec := range decorators {
			isOutermost := i == len(decorators)-1
			if isOutermost {
				base.Constructor = dec.Constructor.Clone()
				base.Type = dec.Type
				base.Alias = nil
				r.rewriteInnerArgs(base.Constructor, prev)
				break
			}

			internalID := uniqueInternalID(usedIDs, "decorator", dec.ID)
			internal := cloneService(dec, internalID)
			internal.Public = false
			internal.Shared = false
			internal.Tags = nil
			internal.Alias = nil
			r.rewriteInnerArgs(internal.Constructor, prev)

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
		}
	}

	if err := r.validateInnerArgs(ctx.services); err != nil {
		return err
	}
	r.rebuildOrder(ctx)
	return nil
}

func (r *decoratorResolver) validateInnerArgs(services map[string]*Service) error {
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

func (r *decoratorResolver) rewriteInnerArgs(cons *Constructor, innerSvc *Service) {
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

func (r *decoratorResolver) resolveAliasTarget(svc *Service) *Service {
	for svc != nil && svc.Alias != nil {
		svc = svc.Alias
	}
	return svc
}

func (r *decoratorResolver) detectDecoratorCycles(decoratesByID map[string]string) error {
	visited := make(map[string]bool)
	stack := make(map[string]bool)
	var dfs func(string, []string) error
	dfs = func(id string, path []string) error {
		if id == "" {
			return nil
		}
		if stack[id] {
			cycle := append(path, id)
			return fmt.Errorf("circular decorator chain: %s", joinIDs(cycle))
		}
		if visited[id] {
			return nil
		}
		visited[id] = true
		stack[id] = true

		if baseID, ok := decoratesByID[id]; ok {
			if err := dfs(baseID, append(path, id)); err != nil {
				return err
			}
		}

		stack[id] = false
		return nil
	}

	for id := range decoratesByID {
		if err := dfs(id, nil); err != nil {
			return err
		}
	}
	return nil
}

func (r *decoratorResolver) rebuildOrder(ctx *buildContext) {
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
	clone.Alias = nil
	clone.Dependencies = nil
	clone.CanError = false
	clone.BuildCanError = false
	if src.Constructor != nil {
		clone.Constructor = src.Constructor.Clone()
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
