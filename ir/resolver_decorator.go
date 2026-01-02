package ir

import (
	"fmt"
	"sort"
)

// decoratorResolver links decorators to their base services
type decoratorResolver struct{}

type decoratorResolverState struct {
	decoratorToInner    map[string]string
	serviceToDecorators map[string][]*Service
}

func (ds *decoratorResolverState) popNext() (innerID string, decoratorID string, ok bool) {
	for serviceID, decorators := range ds.serviceToDecorators {
		decoratorID = decorators[0].ID
		delete(ds.decoratorToInner, decoratorID)

		decorators = decorators[1:]
		ds.serviceToDecorators[serviceID] = decorators
		if len(decorators) == 0 {
			delete(ds.serviceToDecorators, serviceID)
		}

		return serviceID, decoratorID, true
	}

	return "", "", false
}

func (r *decoratorResolver) buildState(ctx *buildContext) (*decoratorResolverState, error) {
	state := &decoratorResolverState{
		decoratorToInner:    make(map[string]string),
		serviceToDecorators: make(map[string][]*Service),
	}

	idToPriorityMap := make(map[string]int)
	for _, svc := range ctx.services {
		cfg := ctx.cfg.Services[svc.ID]
		if cfg.Decorates == "" {
			continue
		}

		base, ok := ctx.services[cfg.Decorates]
		if !ok {
			return nil, fmt.Errorf("decorator %q decorates unknown service %q", svc.ID, cfg.Decorates)
		}
		if baseCfg := ctx.cfg.Services[base.ID]; baseCfg.Decorates != "" {
			return nil, fmt.Errorf("decorator %q cannot be decorated", base.ID)
		}

		state.decoratorToInner[svc.ID] = base.ID
		state.serviceToDecorators[base.ID] = append(state.serviceToDecorators[base.ID], svc)
		idToPriorityMap[svc.ID] = cfg.DecorationPriority
	}

	for baseID, decs := range state.serviceToDecorators {
		if len(decs) > 1 {
			sort.Slice(decs, func(i, j int) bool {
				return idToPriorityMap[decs[i].ID] < idToPriorityMap[decs[j].ID]
			})
			state.serviceToDecorators[baseID] = decs
		}
	}

	return state, nil
}

// resolve links decorators and expands them into plain services and aliases.
func (r *decoratorResolver) resolve(ctx *buildContext) error {
	state, err := r.buildState(ctx)
	if err != nil {
		return err
	}

	if err := r.detectDecoratorCycles(state.decoratorToInner); err != nil {
		return err
	}

	for {
		innerID, decoratorID, ok := state.popNext()

		if !ok {
			break
		}

		if err := r.expandOne(ctx, innerID, decoratorID); err != nil {
			return err
		}
	}

	r.rebuildOrder(ctx)

	return r.validateInnerArgs(ctx.services)
}

func (r *decoratorResolver) rewriteInnerArgs(cons *Constructor, innerSvc *Service) {
	for _, arg := range cons.Args {
		if arg.Kind != InnerArg {
			continue
		}

		arg.Kind = ServiceRefArg
		arg.Service = innerSvc
	}
}

func (r *decoratorResolver) expandOne(ctx *buildContext, innerID, decoratorID string) error {
	innerService := ctx.services[innerID]

	var aliasService *Service

	if innerService.IsAlias() {
		aliasService = innerService
		innerService = innerService.Alias
		aliasService.Alias = nil
	} else {
		aliasService = innerService

		innerService = innerService.Clone()
		innerService.ID = decoratorID + ".inner"

		aliasService.Constructor = nil
		aliasService.Tags = nil
		aliasService.Dependencies = nil
	}

	decoratorService := ctx.services[decoratorID]
	chainShared := aliasService.Shared || innerService.Shared || decoratorService.Shared
	decoratorService.Shared = chainShared
	r.rewriteInnerArgs(decoratorService.Constructor, innerService)

	aliasService.ID = innerID
	aliasService.Type = decoratorService.Type
	aliasService.Alias = decoratorService
	aliasService.Shared = chainShared

	ctx.services[innerService.ID] = innerService
	ctx.services[aliasService.ID] = aliasService

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
