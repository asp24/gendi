package di

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Pass is a compiler pass that can mutate config before validation and generation.
type Pass interface {
	Name() string
	Process(cfg *Config) error
}

// Config is the root configuration for the DI container.
type Config struct {
	Imports    []string          `yaml:"imports"`
	Parameters map[string]Parameter `yaml:"parameters"`
	Tags       map[string]Tag       `yaml:"tags"`
	Services   map[string]*Service  `yaml:"services"`
}

// Parameter defines a typed parameter literal.
type Parameter struct {
	Type  string    `yaml:"type"`
	Value yaml.Node `yaml:"value"`
}

// Tag defines a tag declaration.
type Tag struct {
	ElementType string `yaml:"element_type"`
	SortBy      string `yaml:"sort_by"`
}

// ServiceTag defines a tag assigned to a service.
type ServiceTag struct {
	Name       string                 `yaml:"name"`
	Attributes map[string]interface{} `yaml:"attributes"`
}

// Service defines a service entry.
type Service struct {
	Type               string       `yaml:"type"`
	Constructor        Constructor  `yaml:"constructor"`
	Shared             *bool        `yaml:"shared"`
	Public             *bool        `yaml:"public"`
	Decorates          string       `yaml:"decorates"`
	DecorationPriority int          `yaml:"decoration_priority"`
	Tags               []ServiceTag `yaml:"tags"`
}

// Constructor defines service constructor configuration.
type Constructor struct {
	Func   string     `yaml:"func"`
	Method string     `yaml:"method"`
	Args   []Argument `yaml:"args"`
}

// ArgumentKind is the parsed kind of a constructor argument.
type ArgumentKind int

const (
	ArgLiteral ArgumentKind = iota
	ArgServiceRef
	ArgInner
	ArgParam
	ArgTagged
)

// Argument represents a constructor argument.
type Argument struct {
	Kind    ArgumentKind
	Value   string
	Literal yaml.Node
}

// UnmarshalYAML parses argument syntax.
func (a *Argument) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.ScalarNode {
		return fmt.Errorf("argument must be a scalar")
	}
	if node.Tag == "!!str" {
		s := node.Value
		switch {
		case s == "@.inner":
			a.Kind = ArgInner
			a.Value = s
			return nil
		case len(s) > 1 && s[0] == '@':
			a.Kind = ArgServiceRef
			a.Value = s[1:]
			return nil
		case len(s) > 2 && s[0] == '%' && s[len(s)-1] == '%':
			a.Kind = ArgParam
			a.Value = s[1 : len(s)-1]
			return nil
		case len(s) > len("!tagged:") && s[:len("!tagged:")] == "!tagged:":
			a.Kind = ArgTagged
			a.Value = s[len("!tagged:"):]
			return nil
		}
	}

	a.Kind = ArgLiteral
	a.Literal = *node
	return nil
}

// LoadConfig loads config with imports resolved.
func LoadConfig(path string) (*Config, error) {
	visited := map[string]bool{}
	return loadConfig(path, visited)
}

func loadConfig(path string, visited map[string]bool) (*Config, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if visited[abs] {
		return nil, fmt.Errorf("cyclic import detected at %s", abs)
	}
	visited[abs] = true

	data, err := readFile(abs)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", abs, err)
	}

	merged := &Config{
		Parameters: map[string]Parameter{},
		Tags:       map[string]Tag{},
		Services:   map[string]*Service{},
	}

	baseDir := filepath.Dir(abs)
	for _, imp := range cfg.Imports {
		impPath := imp
		if !filepath.IsAbs(impPath) {
			impPath = filepath.Join(baseDir, imp)
		}
		child, err := loadConfig(impPath, visited)
		if err != nil {
			return nil, err
		}
		merged = mergeConfig(merged, child)
	}

	cfg.Imports = nil
	merged = mergeConfig(merged, cfg)
	return merged, nil
}

func mergeConfig(dst, src *Config) *Config {
	if dst.Parameters == nil {
		dst.Parameters = map[string]Parameter{}
	}
	if dst.Tags == nil {
		dst.Tags = map[string]Tag{}
	}
	if dst.Services == nil {
		dst.Services = map[string]*Service{}
	}

	for k, v := range src.Parameters {
		dst.Parameters[k] = v
	}
	for k, v := range src.Tags {
		dst.Tags[k] = v
	}
	for k, v := range src.Services {
		copySvc := *v
		dst.Services[k] = &copySvc
	}
	return dst
}

func readFile(path string) ([]byte, error) {
	return osReadFile(path)
}

// osReadFile is defined in config_os.go to ease testing.
var osReadFile = readFileDefault

func readFileDefault(path string) ([]byte, error) {
	return os.ReadFile(path)
}
