package yaml

import (
	"fmt"
	"os"
	"path/filepath"

	ylib "gopkg.in/yaml.v3"

	di "github.com/asp24/gendi"
)

// ConfigLoaderYaml loads YAML configuration files with import resolution.
type ConfigLoaderYaml struct {
	resolver *di.ImportResolver
	parser   *Parser
}

// NewConfigLoaderYaml creates a new YAML config loader with dependencies.
func NewConfigLoaderYaml(resolver *di.ImportResolver, parser *Parser) *ConfigLoaderYaml {
	return &ConfigLoaderYaml{
		resolver: resolver,
		parser:   parser,
	}
}

// Load loads a YAML config file with imports resolved.
func (l *ConfigLoaderYaml) Load(path string) (*di.Config, error) {
	visited := map[string]bool{}
	return l.loadRecursive(path, visited)
}

func (l *ConfigLoaderYaml) loadRecursive(path string, visited map[string]bool) (*di.Config, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if visited[abs] {
		return nil, fmt.Errorf("cyclic import detected at %s", abs)
	}
	visited[abs] = true

	data, err := l.readFile(abs)
	if err != nil {
		return nil, err
	}

	raw, err := l.parseRaw(data)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", abs, err)
	}

	merged := di.NewConfig()

	baseDir := filepath.Dir(abs)
	for _, imp := range raw.Imports {
		impPaths, err := l.resolver.Resolve(baseDir, imp.Path)
		if err != nil {
			return nil, fmt.Errorf("resolve import %q: %w", imp.Path, err)
		}
		for _, impPath := range impPaths {
			child, err := l.loadRecursive(impPath, visited)
			if err != nil {
				return nil, err
			}
			merged = merged.MergeWith(child)
		}
	}

	cfg, err := l.parser.convertConfig(raw)
	if err != nil {
		return nil, fmt.Errorf("convert %s: %w", abs, err)
	}

	return merged.MergeWith(cfg), nil
}

// parseRaw parses YAML data into raw config (with imports).
func (l *ConfigLoaderYaml) parseRaw(data []byte) (*RawConfig, error) {
	var raw RawConfig
	if err := l.yamlUnmarshal(data, &raw); err != nil {
		return nil, err
	}
	return &raw, nil
}

func (l *ConfigLoaderYaml) readFile(path string) ([]byte, error) {
	return l.osReadFile(path)
}

// osReadFile can be replaced for testing.
var defaultOsReadFile = os.ReadFile

func (l *ConfigLoaderYaml) osReadFile(path string) ([]byte, error) {
	return defaultOsReadFile(path)
}

// yamlUnmarshal wraps yaml.Unmarshal for testability.
var defaultYamlUnmarshal = ylib.Unmarshal

func (l *ConfigLoaderYaml) yamlUnmarshal(data []byte, v interface{}) error {
	return defaultYamlUnmarshal(data, v)
}
