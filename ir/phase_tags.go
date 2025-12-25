package ir

import "fmt"

// tagPhase builds tags from config
type tagPhase struct{}

// build converts config tags to IR tags
func (p *tagPhase) build(ctx *buildContext) error {
	for name, tag := range ctx.cfg.Tags {
		if tag.ElementType == "" {
			return fmt.Errorf("tag %q missing element_type", name)
		}
		elemType, err := ctx.resolver.LookupType(tag.ElementType)
		if err != nil {
			return fmt.Errorf("tag %q element_type: %w", name, err)
		}
		ctx.tags[name] = &Tag{
			Name:        name,
			ElementType: elemType,
			SortBy:      tag.SortBy,
			Services:    []*Service{},
		}
	}
	return nil
}
