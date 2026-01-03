package ir

import (
	"fmt"
	"strings"

	di "github.com/asp24/gendi"
)

// servicePhase initializes services from config
type servicePhase struct{}

// build initializes IR services and determines service order
func (p *servicePhase) build(cfg *di.Config, container *Container) error {
	for id, svc := range cfg.Services {
		// Validate service ID is not empty or whitespace-only
		if id == "" {
			return fmt.Errorf("service ID cannot be empty")
		}
		if strings.TrimSpace(id) == "" {
			return fmt.Errorf("service ID %q cannot be whitespace-only", id)
		}

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
			tag, ok := container.Tags[st.Name]
			if !ok {
				// Create tag on-demand - ElementType will be inferred later
				tag = &Tag{
					Name:     st.Name,
					Services: []*Service{},
				}
				container.Tags[st.Name] = tag
			}
			irSvc.Tags = append(irSvc.Tags, &ServiceTag{
				Tag:        tag,
				Attributes: st.Attributes,
			})
		}

		container.Services[id] = irSvc
	}

	return nil
}
