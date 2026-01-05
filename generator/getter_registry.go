package generator

import (
	"fmt"

	"github.com/asp24/gendi/ir"
	"github.com/asp24/gendi/xmaps"
)

// getterRegistry manages unique getter names for services and tags.
type getterRegistry struct {
	ident *identGenerator

	publicService  map[string]string
	privateService map[string]string
	publicTag      map[string]string
	privateTag     map[string]string
}

// newGetterRegistry creates a new getter registry.
func newGetterRegistry(ident *identGenerator) *getterRegistry {
	return &getterRegistry{
		ident:          ident,
		publicService:  make(map[string]string),
		privateService: make(map[string]string),
		publicTag:      make(map[string]string),
		privateTag:     make(map[string]string),
	}
}

// Assign assigns unique getter names for all services and tags.
func (gr *getterRegistry) Assign(orderedServiceIDs []string, services map[string]*serviceDef, tags map[string]*ir.Tag, privateTagNames []string) {
	// Assign public getter names
	used := map[string]bool{}
	for _, id := range orderedServiceIDs {
		if services[id].public {
			base := gr.ident.Getter(id, true)
			name := gr.uniqueName(base, used)
			used[name] = true
			gr.publicService[id] = name
		}
	}

	// Assign public tag getter names
	if len(tags) > 0 {
		for _, name := range xmaps.OrderedKeys(tags) {
			if !tags[name].Public {
				continue
			}
			base := gr.ident.TagGetter(name, true)
			getter := gr.uniqueName(base, used)
			used[getter] = true
			gr.publicTag[name] = getter
		}
	}

	// Assign private getter names
	privateUsed := map[string]bool{}
	for _, id := range orderedServiceIDs {
		base := gr.ident.Getter(id, false)
		name := gr.uniqueName(base, privateUsed)
		privateUsed[name] = true
		gr.privateService[id] = name
	}

	// Assign private tag getter names
	for _, name := range privateTagNames {
		base := gr.ident.TagGetter(name, false)
		getter := gr.uniqueName(base, privateUsed)
		privateUsed[getter] = true
		gr.privateTag[name] = getter
	}
}

// PublicService returns the public getter name for a service.
func (gr *getterRegistry) PublicService(id string) string {
	return gr.publicService[id]
}

// PrivateService returns the private getter name for a service.
func (gr *getterRegistry) PrivateService(id string) string {
	return gr.privateService[id]
}

// PublicTag returns the public getter name for a tag.
func (gr *getterRegistry) PublicTag(name string) string {
	return gr.publicTag[name]
}

// PrivateTag returns the private getter name for a tag.
func (gr *getterRegistry) PrivateTag(name string) string {
	return gr.privateTag[name]
}

// uniqueName generates a unique name by appending numbers if needed.
func (gr *getterRegistry) uniqueName(base string, used map[string]bool) string {
	name := base
	if used[name] {
		for i := 2; ; i++ {
			candidate := fmt.Sprintf("%s%d", base, i)
			if !used[candidate] {
				name = candidate
				break
			}
		}
	}
	return name
}
