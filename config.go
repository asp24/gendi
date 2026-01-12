package di

import "fmt"

// Pass is a compiler pass that transforms config before validation and generation.
// Passes mutate the config and return it for chaining.
type Pass interface {
	Name() string
	Process(cfg *Config) (*Config, error)
}

// ApplyPasses applies compiler passes sequentially to the config.
// Each pass receives the result of the previous pass.
func ApplyPasses(cfg *Config, passes []Pass) (*Config, error) {
	result := cfg
	for _, pass := range passes {
		transformed, err := pass.Process(result)
		if err != nil {
			return nil, fmt.Errorf("pass %q failed: %w", pass.Name(), err)
		}
		result = transformed
	}
	return result, nil
}

// ApplyInternalPasses applies mandatory internal transformation passes.
// These passes desugar high-level config constructs (like decorators) into
// simpler primitives before IR building.
func ApplyInternalPasses(cfg *Config) (*Config, error) {
	return ApplyPasses(cfg, []Pass{
		&DecoratorPass{},
	})
}

// Config is the root configuration for the DI container.
// This is a resolved configuration with no import directives.
type Config struct {
	Parameters map[string]Parameter
	Tags       map[string]Tag
	Services   map[string]Service
}

func NewConfig() *Config {
	return &Config{
		Parameters: make(map[string]Parameter),
		Tags:       make(map[string]Tag),
		Services:   make(map[string]Service),
	}
}

// MergeWith merges src into cfg and returns cfg.
func (cfg *Config) MergeWith(src *Config) *Config {
	if src == nil {
		return cfg
	}

	for k, v := range src.Parameters {
		cfg.Parameters[k] = v
	}
	for k, v := range src.Tags {
		cfg.Tags[k] = v
	}
	for k, v := range src.Services {
		cfg.Services[k] = v
	}
	return cfg
}

// Parameter defines a typed parameter literal.
type Parameter struct {
	Type  string
	Value Literal
}

// Tag defines a tag declaration.
type Tag struct {
	ElementType string
	SortBy      string
	Public      bool
	Auto        bool
}

// ServiceTag defines a tag assigned to a service.
type ServiceTag struct {
	Name       string
	Attributes map[string]interface{}
}

// Service defines a service entry.
type Service struct {
	Type               string
	Constructor        Constructor
	Shared             bool
	Public             bool
	Decorates          string
	DecorationPriority int
	Tags               []ServiceTag
	Alias              string
}

// Constructor defines service constructor configuration.
type Constructor struct {
	Func   string
	Method string
	Args   []Argument
}

// ArgumentKind is the parsed kind of a constructor argument.
type ArgumentKind int

const (
	ArgLiteral ArgumentKind = iota
	ArgServiceRef
	ArgInner
	ArgParam
	ArgTagged
	ArgSpread
)

// Argument represents a constructor argument.
type Argument struct {
	Kind    ArgumentKind
	Value   string
	Literal Literal
}
