package ir

import (
	"fmt"

	di "github.com/asp24/gendi"
)

// tagPhase builds tags from config
type tagPhase struct {
	resolver TypeResolver
}

// build converts config tags to IR tags
func (p *tagPhase) build(cfg *di.Config, container *Container) error {
	for name, tag := range cfg.Tags {
		if tag.Public && tag.ElementType == "" {
			return fmt.Errorf("tag %q public requires element_type", name)
		}
		irTag := &Tag{
			Name:     name,
			SortBy:   tag.SortBy,
			Public:   tag.Public,
			Services: []*Service{},
		}

		// ElementType is now optional - can be inferred from constructor arguments
		if tag.ElementType != "" {
			elemType, err := p.resolver.LookupType(tag.ElementType)
			if err != nil {
				return fmt.Errorf("tag %q element_type: %w", name, err)
			}
			irTag.ElementType = elemType
		}

		container.tags[name] = irTag
	}
	return nil
}
