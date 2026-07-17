package generator

import (
	"fmt"
)

// GetterRegistry allocates unique method and field identifiers for services.
// Sanitizing service IDs is many-to-one (".", "-", "_" and case fold
// together), so every identifier derived from a service ID — getters, build
// methods and container struct fields — must be allocated through this
// registry instead of being recomputed from the raw ID.
// Note: After tag desugaring, tags become regular services and are handled
// by the service getter methods. The IdentGenerator.Getter method detects
// !tagged: prefix services and generates appropriate TagGetter style names.
type GetterRegistry struct {
	identGenerator *IdentGenerator

	publicService     map[string]string
	mustPublicService map[string]string
	privateService    map[string]string
	buildFunc         map[string]string
	field             map[string]string
}

// NewGetterRegistry creates a new getter registry.
func NewGetterRegistry(identGenerator *IdentGenerator) *GetterRegistry {
	return &GetterRegistry{
		identGenerator:    identGenerator,
		publicService:     make(map[string]string),
		mustPublicService: make(map[string]string),
		privateService:    make(map[string]string),
		buildFunc:         make(map[string]string),
		field:             make(map[string]string),
	}
}

// uniqueName generates a unique name by appending numbers if needed.
// The used map is keyed by allocated name and holds the owning service ID.
func (gr *GetterRegistry) uniqueName(base string, used map[string]string) string {
	name := base
	for i := 2; ; i++ {
		if _, taken := used[name]; !taken {
			return name
		}

		name = fmt.Sprintf("%s%d", base, i)
	}
}

// uniqueFieldName generates a unique struct field name whose "<name>Init"
// companion is also free: value-typed shared services render the companion
// as a separate flag field.
func (gr *GetterRegistry) uniqueFieldName(base string, used map[string]string) string {
	name := base
	for i := 2; ; i++ {
		_, nameTaken := used[name]
		_, initTaken := used[name+"Init"]
		if !nameTaken && !initTaken {
			return name
		}

		name = fmt.Sprintf("%s%d", base, i)
	}
}

func (gr *GetterRegistry) assignOrError(name, serviceID string, used map[string]string) error {
	if owner, taken := used[name]; taken {
		return fmt.Errorf("duplicate identifier %s: services %q and %q map to the same name", name, owner, serviceID)
	}

	used[name] = serviceID

	return nil
}

// Assign assigns unique getter, build method and field names for all services.
func (gr *GetterRegistry) Assign(orderedServiceIDs []string, services map[string]*serviceDef) error {
	// Assign public getter names
	used := map[string]string{}
	for _, id := range orderedServiceIDs {
		if !services[id].public {
			continue
		}

		{
			getter := gr.identGenerator.Getter(id, true)
			if err := gr.assignOrError(getter, id, used); err != nil {
				return err
			}

			gr.publicService[id] = getter
		}

		{
			getter := gr.identGenerator.Must(id)
			if err := gr.assignOrError(getter, id, used); err != nil {
				return err
			}

			gr.mustPublicService[id] = getter
		}
	}

	// Assign private getter names
	privateUsed := map[string]string{}
	for _, id := range orderedServiceIDs {
		base := gr.identGenerator.Getter(id, false)
		name := gr.uniqueName(base, privateUsed)
		privateUsed[name] = id
		gr.privateService[id] = name
	}

	// Assign build method names (aliases delegate to their target and have
	// no build method).
	buildUsed := map[string]string{}
	for _, id := range orderedServiceIDs {
		if services[id].IsAlias() {
			continue
		}

		name := gr.uniqueName(gr.identGenerator.Build(id), buildUsed)
		buildUsed[name] = id
		gr.buildFunc[id] = name
	}

	// Assign container struct field names (only shared non-alias services
	// cache their instance in a field). Reserve the "<field>Init" companion
	// as well: value-typed services render it as a separate flag field.
	fieldUsed := map[string]string{}
	for _, id := range orderedServiceIDs {
		if services[id].IsAlias() || !services[id].shared {
			continue
		}

		name := gr.uniqueFieldName(gr.identGenerator.Field(id), fieldUsed)
		fieldUsed[name] = id
		fieldUsed[name+"Init"] = id
		gr.field[id] = name
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

// BuildFunc returns the build method name for a service.
func (gr *GetterRegistry) BuildFunc(id string) string {
	return gr.buildFunc[id]
}

// Field returns the container struct field name for a shared service.
func (gr *GetterRegistry) Field(id string) string {
	return gr.field[id]
}
