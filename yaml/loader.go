package yaml

import (
	di "github.com/asp24/gendi"
)

// LoadConfig loads a YAML config file with imports resolved.
// This is a convenience function that creates a ConfigLoaderYaml with default dependencies.
func LoadConfig(path string) (*di.Config, error) {
	resolver := NewImportResolver()
	merger := NewConfigMerger()
	parser := NewParser()
	loader := NewConfigLoaderYaml(resolver, merger, parser)
	return loader.Load(path)
}
