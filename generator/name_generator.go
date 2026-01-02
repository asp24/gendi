package generator

import (
	"fmt"
	"sort"
	"strings"

	"github.com/asp24/gendi/ir"
)

// nameGenerator encapsulates all naming logic for generated code
type nameGenerator struct {
	publicGetterNames     map[string]string // service ID -> public getter name
	privateGetterNames    map[string]string // service ID -> private getter name
	publicTagGetterNames  map[string]string // tag name -> public getter name
	privateTagGetterNames map[string]string // tag name -> private getter name
}

// newNameGenerator creates a new name generator
func newNameGenerator() *nameGenerator {
	return &nameGenerator{
		publicGetterNames:     make(map[string]string),
		privateGetterNames:    make(map[string]string),
		publicTagGetterNames:  make(map[string]string),
		privateTagGetterNames: make(map[string]string),
	}
}

// assignGetterNames assigns unique getter names for all services
func (ng *nameGenerator) assignGetterNames(orderedServiceIDs []string, services map[string]*serviceDef, tags map[string]*ir.Tag) {
	// Assign public getter names
	used := map[string]bool{}
	for _, id := range orderedServiceIDs {
		if services[id].public {
			base := "Get" + ng.toCamel(id)
			name := ng.uniqueName(base, used)
			used[name] = true
			ng.publicGetterNames[id] = name
		}
	}

	// Assign public tag getter names
	if len(tags) > 0 {
		tagNames := make([]string, 0, len(tags))
		for name := range tags {
			tagNames = append(tagNames, name)
		}
		sort.Strings(tagNames)
		for _, name := range tagNames {
			if !tags[name].Public {
				continue
			}
			base := "GetTaggedWith" + ng.toCamel(name)
			getter := ng.uniqueName(base, used)
			used[getter] = true
			ng.publicTagGetterNames[name] = getter
		}
	}

	// Assign private getter names
	privateUsed := map[string]bool{}
	for _, id := range orderedServiceIDs {
		base := "get" + ng.toCamel(id)
		name := ng.uniqueName(base, privateUsed)
		privateUsed[name] = true
		ng.privateGetterNames[id] = name
	}

	// Assign private tag getter names
	if len(tags) > 0 {
		tagNames := make([]string, 0, len(tags))
		for name := range tags {
			tagNames = append(tagNames, name)
		}
		sort.Strings(tagNames)
		for _, name := range tagNames {
			if !tags[name].Public {
				continue
			}
			base := "getTaggedWith" + ng.toCamel(name)
			getter := ng.uniqueName(base, privateUsed)
			privateUsed[getter] = true
			ng.privateTagGetterNames[name] = getter
		}
	}
}

// uniqueName generates a unique name by appending numbers if needed
func (ng *nameGenerator) uniqueName(base string, used map[string]bool) string {
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

// publicGetterName returns the public getter name for a service
func (ng *nameGenerator) publicGetterName(id string) string {
	return ng.publicGetterNames[id]
}

// publicTagGetterName returns the public getter name for a tag.
func (ng *nameGenerator) publicTagGetterName(tag string) string {
	return ng.publicTagGetterNames[tag]
}

// privateTagGetterName returns the private getter name for a tag.
func (ng *nameGenerator) privateTagGetterName(tag string) string {
	return ng.privateTagGetterNames[tag]
}

// privateGetterName returns the private getter name for a service
func (ng *nameGenerator) privateGetterName(id string) string {
	return ng.privateGetterNames[id]
}

// toCamel converts a service ID to CamelCase
func (ng *nameGenerator) toCamel(id string) string {
	parts := strings.FieldsFunc(id, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9')
	})
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	if len(parts) == 0 {
		return "Service"
	}
	return strings.Join(parts, "")
}

// fieldIdent returns the field identifier for a service
func (ng *nameGenerator) fieldIdent(id string) string {
	return "svc_" + ng.sanitizeIdent(id)
}

// paramIdent returns the identifier for a parameter
func (ng *nameGenerator) paramIdent(name string) string {
	return "param_" + ng.sanitizeIdent(name)
}

// varIdent returns a variable identifier with a prefix
func (ng *nameGenerator) varIdent(prefix, id string) string {
	return prefix + "_" + ng.sanitizeIdent(id)
}

// sanitizeIdent converts a string to a valid Go identifier
func (ng *nameGenerator) sanitizeIdent(id string) string {
	var b strings.Builder
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	if b.Len() == 0 {
		return "id"
	}
	return b.String()
}

// buildName returns the build function name for a service
func (ng *nameGenerator) buildName(svc *serviceDef) string {
	return "build" + ng.toCamel(svc.id)
}

// decoratorBuildName returns the decorator build function name
func (ng *nameGenerator) decoratorBuildName(svc *serviceDef) string {
	return "build" + ng.toCamel(svc.id) + "Decorator"
}

// chainBuildName returns the decorator chain build function name
func (ng *nameGenerator) chainBuildName(svc *serviceDef) string {
	return "buildDecorated" + ng.toCamel(svc.id)
}
