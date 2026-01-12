package ir

import (
	"fmt"
	"go/types"

	di "github.com/asp24/gendi"
	"github.com/asp24/gendi/xmaps"
)

// taggedServiceResolver links services to their tags and validates type compatibility.
// This phase must run after constructorResolver (which sets service types) and before
// tagDesugarPhase (which needs tag.Services populated).
type taggedServiceResolver struct{}

// resolve links services to their tags and validates type compatibility
func (r *taggedServiceResolver) resolve(_ *di.Config, container *Container) error {
	for _, id := range xmaps.OrderedKeys(container.Services) {
		svc := container.Services[id]
		for _, st := range svc.Tags {
			// Validate service type is assignable to tag's ElementType (if known)
			if st.Tag.ElementType != nil && svc.Type != nil {
				if !types.AssignableTo(svc.Type, st.Tag.ElementType) {
					return fmt.Errorf("service %q with tag %q: type %s is not assignable to %s",
						svc.ID, st.Tag.Name, svc.Type, st.Tag.ElementType)
				}
			}
			st.Tag.Services = append(st.Tag.Services, svc)
		}
	}
	return nil
}
