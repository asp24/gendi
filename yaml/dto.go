package yaml

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	di "github.com/asp24/gendi"
)

// RawConfig is the YAML-specific representation of a config file.
type RawConfig struct {
	Imports    []rawImport             `yaml:"imports"`
	Parameters map[string]RawParameter `yaml:"parameters"`
	Tags       map[string]RawTag       `yaml:"tags"`
	Services   map[string]*RawService  `yaml:"services"`
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

type RawParameter struct {
	Type  string    `yaml:"type"`
	Value yaml.Node `yaml:"value"`
}

type RawTag struct {
	ElementType string `yaml:"element_type"`
	SortBy      string `yaml:"sort_by"`
}

type rawServiceTag struct {
	Name       string                 `yaml:"name"`
	Attributes map[string]interface{} `yaml:"attributes"`
}

type RawService struct {
	Type               string          `yaml:"type"`
	Constructor        RawConstructor  `yaml:"constructor"`
	Shared             *bool           `yaml:"shared"`
	Public             bool            `yaml:"public,omitempty"`
	Decorates          string          `yaml:"decorates"`
	DecorationPriority int             `yaml:"decoration_priority"`
	Tags               []rawServiceTag `yaml:"tags"`
	Alias              string          `yaml:"alias"`
}

func (s *RawService) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		var ref string
		if err := node.Decode(&ref); err != nil {
			return err
		}
		if !strings.HasPrefix(ref, "@") || len(ref) == 1 {
			return fmt.Errorf("service alias must start with @")
		}
		*s = RawService{Alias: ref[1:]}
		return nil
	case yaml.MappingNode:
		type alias RawService
		var decoded alias
		if err := node.Decode(&decoded); err != nil {
			return err
		}
		*s = RawService(decoded)
		return nil
	default:
		return fmt.Errorf("service must be a mapping or alias")
	}
}

type RawConstructor struct {
	Func   string        `yaml:"func"`
	Method string        `yaml:"method"`
	Args   []RawArgument `yaml:"args"`
}

func (c *RawConstructor) UnmarshalYAML(node *yaml.Node) error {
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
	c.Args = make([]RawArgument, len(decoded.Args))
	for i := range decoded.Args {
		if err := c.Args[i].UnmarshalYAML(&decoded.Args[i]); err != nil {
			return err
		}
	}
	return nil
}

type RawArgument struct {
	kind    di.ArgumentKind
	value   string
	literal yaml.Node
}

func (a *RawArgument) UnmarshalYAML(node *yaml.Node) error {
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
