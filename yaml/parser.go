// Package yaml provides YAML parsing for DI configuration files.
package yaml

import (
	"fmt"
	"strings"

	di "github.com/asp24/gendi"
	"gopkg.in/yaml.v3"
)


// rawConfig is the YAML-specific representation of a config file.
type rawConfig struct {
	Imports    []rawImport             `yaml:"imports"`
	Parameters map[string]rawParameter `yaml:"parameters"`
	Tags       map[string]rawTag       `yaml:"tags"`
	Services   map[string]*rawService  `yaml:"services"`
}

type rawImport struct {
	Path   string `yaml:"path"`
	Prefix string `yaml:"prefix"`
}

func (i *rawImport) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		var path string
		if err := node.Decode(&path); err != nil {
			return err
		}
		i.Path = path
		return nil
	case yaml.MappingNode:
		type alias rawImport
		var decoded alias
		if err := node.Decode(&decoded); err != nil {
			return err
		}
		if decoded.Path == "" {
			return fmt.Errorf("import path is required")
		}
		*i = rawImport(decoded)
		return nil
	default:
		return fmt.Errorf("import must be a string or mapping")
	}
}

type rawParameter struct {
	Type  string    `yaml:"type"`
	Value yaml.Node `yaml:"value"`
}

type rawTag struct {
	ElementType string `yaml:"element_type"`
	SortBy      string `yaml:"sort_by"`
}

type rawServiceTag struct {
	Name       string                 `yaml:"name"`
	Attributes map[string]interface{} `yaml:"attributes"`
}

type rawService struct {
	Type               string          `yaml:"type"`
	Constructor        rawConstructor  `yaml:"constructor"`
	Shared             *bool           `yaml:"shared"`
	Public             bool            `yaml:"public,omitempty"`
	Decorates          string          `yaml:"decorates"`
	DecorationPriority int             `yaml:"decoration_priority"`
	Tags               []rawServiceTag `yaml:"tags"`
	Alias              string          `yaml:"alias"`
}

func (s *rawService) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		var ref string
		if err := node.Decode(&ref); err != nil {
			return err
		}
		if !strings.HasPrefix(ref, "@") || len(ref) == 1 {
			return fmt.Errorf("service alias must start with @")
		}
		*s = rawService{Alias: ref[1:]}
		return nil
	case yaml.MappingNode:
		type alias rawService
		var decoded alias
		if err := node.Decode(&decoded); err != nil {
			return err
		}
		*s = rawService(decoded)
		return nil
	default:
		return fmt.Errorf("service must be a mapping or alias")
	}
}

type rawConstructor struct {
	Func   string        `yaml:"func"`
	Method string        `yaml:"method"`
	Args   []rawArgument `yaml:"args"`
}

func (c *rawConstructor) UnmarshalYAML(node *yaml.Node) error {
	type raw struct {
		Func   string      `yaml:"func"`
		Method string      `yaml:"method"`
		Args   []yaml.Node `yaml:"args"`
	}
	var decoded raw
	if err := node.Decode(&decoded); err != nil {
		return err
	}
	c.Func = decoded.Func
	c.Method = decoded.Method
	if len(decoded.Args) == 0 {
		return nil
	}
	c.Args = make([]rawArgument, len(decoded.Args))
	for i := range decoded.Args {
		if err := c.Args[i].UnmarshalYAML(&decoded.Args[i]); err != nil {
			return err
		}
	}
	return nil
}

type rawArgument struct {
	kind    di.ArgumentKind
	value   string
	literal yaml.Node
}

func (a *rawArgument) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.ScalarNode {
		return fmt.Errorf("argument must be a scalar")
	}
	if node.Tag == "!!str" {
		s := node.Value
		switch {
		case s == "@.inner":
			a.kind = di.ArgInner
			a.value = s
			return nil
		case len(s) > 1 && s[0] == '@':
			a.kind = di.ArgServiceRef
			a.value = s[1:]
			return nil
		case len(s) > 2 && s[0] == '%' && s[len(s)-1] == '%':
			a.kind = di.ArgParam
			a.value = s[1 : len(s)-1]
			return nil
		case len(s) > len("!tagged:") && s[:len("!tagged:")] == "!tagged:":
			a.kind = di.ArgTagged
			a.value = s[len("!tagged:"):]
			return nil
		}
	}

	a.kind = di.ArgLiteral
	a.literal = *node
	return nil
}

// Parse parses YAML data into a di.Config.
func Parse(data []byte) (*di.Config, error) {
	var raw rawConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	return convertConfig(&raw)
}

func convertConfig(raw *rawConfig) (*di.Config, error) {
	cfg := &di.Config{
		Parameters: make(map[string]di.Parameter),
		Tags:       make(map[string]di.Tag),
		Services:   make(map[string]*di.Service),
	}

	// Convert parameters
	for name, param := range raw.Parameters {
		lit, err := convertLiteral(&param.Value)
		if err != nil {
			return nil, fmt.Errorf("parameter %q: %w", name, err)
		}
		cfg.Parameters[name] = di.Parameter{
			Type:  param.Type,
			Value: lit,
		}
	}

	// Convert tags
	for name, tag := range raw.Tags {
		cfg.Tags[name] = di.Tag{
			ElementType: tag.ElementType,
			SortBy:      tag.SortBy,
		}
	}

	// Convert services
	for name, svc := range raw.Services {
		converted, err := convertService(svc)
		if err != nil {
			return nil, fmt.Errorf("service %q: %w", name, err)
		}
		cfg.Services[name] = converted
	}

	return cfg, nil
}

func convertService(raw *rawService) (*di.Service, error) {
	svc := &di.Service{
		Type:               raw.Type,
		Shared:             raw.Shared,
		Public:             raw.Public,
		Decorates:          raw.Decorates,
		DecorationPriority: raw.DecorationPriority,
		Alias:              raw.Alias,
	}

	// Convert tags
	svc.Tags = make([]di.ServiceTag, len(raw.Tags))
	for i, tag := range raw.Tags {
		svc.Tags[i] = di.ServiceTag{
			Name:       tag.Name,
			Attributes: tag.Attributes,
		}
	}

	// Convert constructor
	svc.Constructor = di.Constructor{
		Func:   raw.Constructor.Func,
		Method: raw.Constructor.Method,
	}

	if len(raw.Constructor.Args) > 0 {
		svc.Constructor.Args = make([]di.Argument, len(raw.Constructor.Args))
		for i, arg := range raw.Constructor.Args {
			converted, err := convertArgument(&arg)
			if err != nil {
				return nil, fmt.Errorf("arg[%d]: %w", i, err)
			}
			svc.Constructor.Args[i] = converted
		}
	}

	return svc, nil
}

func convertArgument(raw *rawArgument) (di.Argument, error) {
	arg := di.Argument{
		Kind:  raw.kind,
		Value: raw.value,
	}

	if raw.kind == di.ArgLiteral {
		lit, err := convertLiteral(&raw.literal)
		if err != nil {
			return di.Argument{}, err
		}
		arg.Literal = lit
	}

	return arg, nil
}

func convertLiteral(node *yaml.Node) (di.Literal, error) {
	switch node.Tag {
	case "!!str":
		return di.NewStringLiteral(node.Value), nil
	case "!!int":
		var v int64
		if err := node.Decode(&v); err != nil {
			return di.Literal{}, err
		}
		return di.NewIntLiteral(v), nil
	case "!!float":
		var v float64
		if err := node.Decode(&v); err != nil {
			return di.Literal{}, err
		}
		return di.NewFloatLiteral(v), nil
	case "!!bool":
		var v bool
		if err := node.Decode(&v); err != nil {
			return di.Literal{}, err
		}
		return di.NewBoolLiteral(v), nil
	case "!!null":
		return di.NewNullLiteral(), nil
	default:
		return di.Literal{}, fmt.Errorf("unsupported literal type %q", node.Tag)
	}
}
