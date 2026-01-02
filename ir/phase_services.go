package ir

import (
	"fmt"
	"sort"
	"strings"
)

// servicePhase initializes services from config
type servicePhase struct{}

// build initializes IR services and determines service order
func (p *servicePhase) build(ctx *buildContext) error {
	ctx.order = make([]string, 0, len(ctx.cfg.Services))
	for id, svc := range ctx.cfg.Services {
		// Validate service ID is not empty or whitespace-only
		if id == "" {
			return fmt.Errorf("service ID cannot be empty")
		}
		if strings.TrimSpace(id) == "" {
			return fmt.Errorf("service ID %q cannot be whitespace-only", id)
		}

		ctx.order = append(ctx.order, id)

		shared := true
		if svc.Shared != nil {
			shared = *svc.Shared
		}
		if svc.Alias != "" {
			shared = false
		}

		irSvc := &Service{
			ID:     id,
			Shared: shared,
			Public: svc.Public,
			Tags:   []*ServiceTag{},
		}

		// Build service tags (create tags on-demand if not declared)
		for _, st := range svc.Tags {
			tag, ok := ctx.tags[st.Name]
			if !ok {
				// Create tag on-demand - ElementType will be inferred later
				tag = &Tag{
					Name:     st.Name,
					Services: []*Service{},
				}
				ctx.tags[st.Name] = tag
			}
			irSvc.Tags = append(irSvc.Tags, &ServiceTag{
				Tag:        tag,
				Attributes: st.Attributes,
			})
		}

		ctx.services[id] = irSvc
	}
	sort.Strings(ctx.order)
	return nil
}
