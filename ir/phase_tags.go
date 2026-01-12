package ir

import (
	"fmt"
	"go/types"

	di "github.com/asp24/gendi"
)

// tagPhase builds tags from config
type tagPhase struct {
	resolver TypeResolver
}

// Apply converts config tags to IR tags
func (p *tagPhase) Apply(cfg *di.Config, container *Container) error {
	for name, tag := range cfg.Tags {
		if tag.Auto {
			if tag.ElementType == "" {
				return fmt.Errorf("tag %q auto requires element_type", name)
			}
			if tag.SortBy != "" {
				return fmt.Errorf("tag %q auto cannot be used with sort_by", name)
			}
		}
		if tag.Public && tag.ElementType == "" {
			return fmt.Errorf("tag %q public requires element_type", name)
		}
		irTag := &Tag{
			Name:     name,
			SortBy:   tag.SortBy,
			Public:   tag.Public,
			Auto:     tag.Auto,
			Services: []*Service{},
		}

		// ElementType is now optional - can be inferred from constructor arguments
		if tag.ElementType != "" {
			elemType, err := p.resolver.LookupType(tag.ElementType)
			if err != nil {
				return fmt.Errorf("tag %q element_type: %w", name, err)
			}
			if tag.Auto {
				if _, ok := elemType.Underlying().(*types.Interface); !ok {
					return fmt.Errorf("tag %q auto element_type must be an interface", name)
				}
			}
			irTag.ElementType = elemType
		}

		container.tags[name] = irTag
	}
	return nil
}
