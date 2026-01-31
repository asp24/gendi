package ir

import (
	"strings"

	di "github.com/asp24/gendi"
	"github.com/asp24/gendi/srcloc"
)

// servicePhase initializes services from config
type servicePhase struct{}

// Apply initializes IR services and determines service order
func (p *servicePhase) Apply(cfg *di.Config, container *Container) error {
	for id, svc := range cfg.Services {
		// Validate service ID is not empty or whitespace-only
		if id == "" {
			return srcloc.Errorf(svc.SourceLoc, "service ID cannot be empty")
		}
		if strings.TrimSpace(id) == "" {
			return srcloc.Errorf(svc.SourceLoc, "service ID %q cannot be whitespace-only", id)
		}

		irSvc := &Service{
			ID:            id,
			Shared:        svc.Shared && svc.Alias == "",
			Public:        svc.Public,
			Autoconfigure: svc.Autoconfigure,
			Tags:          []*ServiceTag{},
		}

		// Build service tags (create tags on-demand if not declared)
		for _, st := range svc.Tags {
			tag, ok := container.tags[st.Name]
			if !ok {
				// Create tag on-demand - ElementType will be inferred later
				tag = &Tag{
					Name:     st.Name,
					Services: []*Service{},
				}
				container.tags[st.Name] = tag
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
