package generator

import (
	"fmt"

	"github.com/asp24/gendi/ir"
	"github.com/asp24/gendi/xmaps"
)

// GetterRegistry manages unique getter names for services and tags.
type GetterRegistry struct {
	identGenerator *IdentGenerator

	publicService     map[string]string
	mustPublicService map[string]string
	privateService    map[string]string
	publicTag         map[string]string
	privateTag        map[string]string
}

// NewGetterRegistry creates a new getter registry.
func NewGetterRegistry(identGenerator *IdentGenerator) *GetterRegistry {
	return &GetterRegistry{
		identGenerator:    identGenerator,
		publicService:     make(map[string]string),
		mustPublicService: make(map[string]string),
		privateService:    make(map[string]string),
		publicTag:         make(map[string]string),
		privateTag:        make(map[string]string),
	}
}

// uniqueName generates a unique name by appending numbers if needed.
func (gr *GetterRegistry) uniqueName(base string, used map[string]bool) string {
	name := base
	for i := 2; ; i++ {
		if !used[name] {
			return name
		}

		name = fmt.Sprintf("%s%d", base, i)
	}
}

func (gr *GetterRegistry) assignOrError(name string, used map[string]bool) error {
	if used[name] {
		return fmt.Errorf("duplicate identifier %s", name)
	}

	used[name] = true

	return nil
}

// Assign assigns unique getter names for all services and tags.
func (gr *GetterRegistry) Assign(orderedServiceIDs []string, services map[string]*serviceDef, tags map[string]*ir.Tag, privateTagNames []string) error {
	// Assign public getter names
	used := map[string]bool{}
	for _, id := range orderedServiceIDs {
		if !services[id].public {
			continue
		}

		{
			getter := gr.identGenerator.Getter(id, true)
			if err := gr.assignOrError(getter, used); err != nil {
				return err
			}

			gr.publicService[id] = getter
		}

		{
			getter := gr.identGenerator.Must(id)
			if err := gr.assignOrError(getter, used); err != nil {
				return err
			}

			gr.mustPublicService[id] = getter
		}
	}

	// Assign public tag getter names
	for _, name := range xmaps.OrderedKeys(tags) {
		if !tags[name].Public {
			continue
		}

		getter := gr.identGenerator.TagGetter(name, true)
		if err := gr.assignOrError(getter, used); err != nil {
			return err
		}

		gr.publicTag[name] = getter
	}

	// Assign private getter names
	privateUsed := map[string]bool{}
	for _, id := range orderedServiceIDs {
		base := gr.identGenerator.Getter(id, false)
		name := gr.uniqueName(base, privateUsed)
		privateUsed[name] = true
		gr.privateService[id] = name
	}

	// Assign private tag getter names
	for _, name := range privateTagNames {
		base := gr.identGenerator.TagGetter(name, false)
		getter := gr.uniqueName(base, privateUsed)
		privateUsed[getter] = true
		gr.privateTag[name] = getter
	}

	return nil
}

// PublicService returns the public getter name for a service.
func (gr *GetterRegistry) PublicService(id string) string {
	return gr.publicService[id]
}

// PrivateService returns the private getter name for a service.
func (gr *GetterRegistry) PrivateService(id string) string {
	return gr.privateService[id]
}

// MustService returns the Must* getter name for a public service.
// It transforms the public getter name (e.g., "GetService") to "MustService".
func (gr *GetterRegistry) MustService(id string) string {
	return gr.mustPublicService[id]
}

// PublicTag returns the public getter name for a tag.
func (gr *GetterRegistry) PublicTag(name string) string {
	return gr.publicTag[name]
}

// PrivateTag returns the private getter name for a tag.
func (gr *GetterRegistry) PrivateTag(name string) string {
	return gr.privateTag[name]
}
