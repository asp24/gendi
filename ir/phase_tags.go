package ir

import (
	"fmt"
	"go/types"

	di "github.com/asp24/gendi"
	"github.com/asp24/gendi/srcloc"
)

// tagPhase builds tags from config
type tagPhase struct {
	resolver TypeResolver
}

// Apply converts config tags to IR tags
func (p *tagPhase) Apply(cfg *di.Config, container *Container) error {
	for name, tag := range cfg.Tags {
		if tag.Autoconfigure {
			if tag.ElementType == "" {
				return srcloc.Errorf(tag.SourceLoc, "tag %q autoconfigure requires element_type", name)
			}
			if tag.SortBy != "" {
				return srcloc.Errorf(tag.SourceLoc, "tag %q autoconfigure cannot be used with sort_by", name)
			}
		}
		if tag.Public && tag.ElementType == "" {
			return srcloc.Errorf(tag.SourceLoc, "tag %q public requires element_type", name)
		}
		irTag := &Tag{
			Name:          name,
			SortBy:        tag.SortBy,
			Public:        tag.Public,
			Autoconfigure: tag.Autoconfigure,
			Services:      []*Service{},
		}

		// ElementType is now optional - can be inferred from constructor arguments
		if tag.ElementType != "" {
			elemType, err := p.resolver.LookupType(tag.ElementType)
			if err != nil {
				return srcloc.WrapError(tag.SourceLoc, fmt.Sprintf("tag %q element_type", name), err)
			}
			if tag.Autoconfigure {
				if _, ok := elemType.Underlying().(*types.Interface); !ok {
					return srcloc.Errorf(tag.SourceLoc, "tag %q autoconfigure element_type must be an interface", name)
				}
			}
			irTag.ElementType = elemType
		}

		container.tags[name] = irTag
	}
	return nil
}
