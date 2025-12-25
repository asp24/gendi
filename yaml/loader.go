package yaml

import (
	di "github.com/asp24/gendi"
	"github.com/asp24/gendi/imprt"
)

// LoadConfig loads a YAML config file with imports resolved.
// This is a convenience function that creates a ConfigLoaderYaml with default dependencies.
func LoadConfig(path string) (*di.Config, error) {
	resolver := imprt.NewResolverCompositeDefault()
	parser := NewParser()
	loader := NewConfigLoaderYaml(resolver, parser)
	return loader.Load(path)
}
