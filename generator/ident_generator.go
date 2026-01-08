package generator

import "strings"

// IdentGenerator generates Go identifiers for generated code.
type IdentGenerator struct{}

func NewIdentGenerator() *IdentGenerator {
	return &IdentGenerator{}
}

// Field returns the field identifier for a service.
func (ig *IdentGenerator) Field(id string) string {
	return "svc_" + ig.sanitize(id)
}

// Var returns a variable identifier with a prefix.
func (ig *IdentGenerator) Var(prefix, id string) string {
	return prefix + "_" + ig.sanitize(id)
}

// Build returns the build function name for a service.
func (ig *IdentGenerator) Build(id string) string {
	return "build" + ig.toCamel(id)
}

// Getter returns a getter method name (e.g., "GetMyService" or "getMyService").
func (ig *IdentGenerator) Getter(id string, public bool) string {
	if public {
		return "Get" + ig.toCamel(id)
	}
	return "get" + ig.toCamel(id)
}

func (ig *IdentGenerator) Must(id string) string {
	return "Must" + ig.toCamel(id)
}

// TagGetter returns a tag getter method name.
func (ig *IdentGenerator) TagGetter(name string, public bool) string {
	if public {
		return "GetTaggedWith" + ig.toCamel(name)
	}
	return "getTaggedWith" + ig.toCamel(name)
}

// toCamel converts a string to CamelCase.
func (ig *IdentGenerator) toCamel(id string) string {
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

// sanitize converts a string to a valid Go identifier.
func (ig *IdentGenerator) sanitize(id string) string {
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
