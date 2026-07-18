package yaml

import (
	di "github.com/gendi-org/gendi"
	"github.com/gendi-org/gendi/imprt"
)

// LoadConfig loads a YAML config file with imports resolved. boundaryRoot is the
// outermost directory imports may not escape, applied to configs that live
// outside any Go module; within a module, imports are confined to that module
// instead. Callers own deriving it — typically the module root of the project
// being generated.
//
// This is a convenience function that creates a ConfigLoaderYaml with default
// dependencies.
func LoadConfig(path, boundaryRoot string) (*di.Config, error) {
	resolver := imprt.NewResolverCompositeDefault(boundaryRoot)
	parser := NewParser()
	loader := NewConfigLoaderYaml(resolver, parser)
	return loader.Load(path)
}
