package yaml

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// RawConfig is the YAML-specific representation of a config file.
type RawConfig struct {
	Imports    []RawImport             `yaml:"imports"`
	Parameters map[string]RawParameter `yaml:"parameters"`
	Tags       map[string]RawTag       `yaml:"tags"`
	Services   map[string]*RawService  `yaml:"services"`
}

type RawImport struct {
	Path string `yaml:"path"`
}

func (i *RawImport) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		var path string
		if err := node.Decode(&path); err != nil {
			return err
		}
		i.Path = path
		return nil
	case yaml.MappingNode:
		type alias RawImport
		var decoded alias
		if err := node.Decode(&decoded); err != nil {
			return err
		}
		if decoded.Path == "" {
			return fmt.Errorf("import path is required")
		}
		*i = RawImport(decoded)
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
	Public      bool   `yaml:"public"`
	Auto        bool   `yaml:"auto"`
}

type RawServiceTag struct {
	Name       string
	Attributes map[string]interface{}
}

func (t *RawServiceTag) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("tag must be a mapping")
	}

	t.Attributes = make(map[string]interface{})

	// Iterate over key-value pairs in the mapping
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		key := keyNode.Value

		if key == "name" {
			// Decode name field
			if err := valueNode.Decode(&t.Name); err != nil {
				return fmt.Errorf("failed to decode tag name: %w", err)
			}
		} else {
			// All other fields are attributes
			var value interface{}
			if err := valueNode.Decode(&value); err != nil {
				return fmt.Errorf("failed to decode tag attribute %q: %w", key, err)
			}
			t.Attributes[key] = value
		}
	}

	if t.Name == "" {
		return fmt.Errorf("tag name is required")
	}

	return nil
}

// ServiceDefaults holds default values for service configuration.
// Only shared and public fields are allowed in _default section.
type ServiceDefaults struct {
	Shared *bool `yaml:"shared"`
	Public *bool `yaml:"public"`
}

type RawService struct {
	Type               string          `yaml:"type"`
	Constructor        RawConstructor  `yaml:"constructor"`
	Shared             *bool           `yaml:"shared"`
	Public             *bool           `yaml:"public"`
	Decorates          string          `yaml:"decorates"`
	DecorationPriority int             `yaml:"decoration_priority"`
	Tags               []RawServiceTag `yaml:"tags"`
	Alias              string          `yaml:"alias"`
}

func (s *RawService) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		var ref string
		if err := node.Decode(&ref); err != nil {
			return err
		}
		*s = RawService{Alias: ref}
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
	Value *string
	Node  *yaml.Node
}

func (a *RawArgument) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode && node.Tag == "!!str" {
		val := node.Value
		a.Value = &val
		return nil
	}
	a.Node = node
	return nil
}
