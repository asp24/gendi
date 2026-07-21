package yaml

import (
	di "github.com/gendi-org/gendi"
	"github.com/gendi-org/gendi/imprt"
)

// LoadConfig loads a YAML config file with imports resolved. boundary is the
// path outside which loading files is forbidden; it applies to configs that
// live outside any Go module — within a module, imports are confined to that
// module instead. Derive it with DefaultBoundary unless a different boundary
// is explicitly required; an empty boundary is an error. moduleContext is the
// directory whose go.mod graph resolves module imports from configs outside
// any Go module; it may be empty when those configs use local imports only.
//
// This is a convenience function that creates a ConfigLoaderYaml with default
// dependencies.
func LoadConfig(path, boundary, moduleContext string) (*di.Config, error) {
	resolver, err := imprt.NewResolver(boundary, moduleContext)
	if err != nil {
		return nil, err
	}
	loader := NewConfigLoaderYaml(resolver, NewParser())
	return loader.Load(path, boundary)
}
