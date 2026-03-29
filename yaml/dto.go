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

func (c *RawConfig) UnmarshalYAML(node *yaml.Node) error {
	// Use type alias to avoid recursion
	type alias RawConfig
	var decoded alias
	if err := node.Decode(&decoded); err != nil {
		return err
	}

	// Manually extract nodes for parameters and tags from the mapping
	if node.Kind == yaml.MappingNode {
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]

			switch keyNode.Value {
			case "parameters":
				// Extract parameter nodes
				if valueNode.Kind == yaml.MappingNode {
					for j := 0; j < len(valueNode.Content); j += 2 {
						paramName := valueNode.Content[j].Value
						paramValueNode := valueNode.Content[j+1]
						if param, ok := decoded.Parameters[paramName]; ok {
							param.Node = paramValueNode
							decoded.Parameters[paramName] = param
						}
					}
				}
			case "tags":
				// Extract tag nodes
				if valueNode.Kind == yaml.MappingNode {
					for j := 0; j < len(valueNode.Content); j += 2 {
						tagName := valueNode.Content[j].Value
						tagValueNode := valueNode.Content[j+1]
						if tag, ok := decoded.Tags[tagName]; ok {
							tag.Node = tagValueNode
							decoded.Tags[tagName] = tag
						}
					}
				}
			}
		}
	}

	*c = RawConfig(decoded)
	return nil
}

type RawImport struct {
	Path    string   `yaml:"path"`
	Exclude []string `yaml:"exclude"`
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
			return nodeErrorf(node, "import path is required")
		}
		*i = RawImport(decoded)
		return nil
	default:
		return nodeErrorf(node, "import must be a string or mapping")
	}
}

type RawParameter struct {
	Type  string    `yaml:"type"`
	Value yaml.Node `yaml:"value"`

	// Node holds the full parameter mapping node for location tracking
	Node *yaml.Node `yaml:"-"`
}

type RawTag struct {
	ElementType   string `yaml:"element_type"`
	SortBy        string `yaml:"sort_by"`
	Public        bool   `yaml:"public"`
	Autoconfigure bool   `yaml:"autoconfigure"`

	// Node holds the tag mapping node for location tracking
	Node *yaml.Node `yaml:"-"`
}

type RawServiceTag struct {
	Name       string
	Attributes map[string]interface{}

	// Node holds the tag mapping node for location tracking
	Node *yaml.Node `yaml:"-"`
}

func (t *RawServiceTag) UnmarshalYAML(node *yaml.Node) error {
	// Preserve node for location tracking
	t.Node = node
	t.Attributes = make(map[string]interface{})

	if node.Kind == yaml.ScalarNode {
		var name string
		if err := node.Decode(&name); err != nil {
			return wrapNodeError(node, "failed to decode tag name", err)
		}
		if name == "" {
			return nodeErrorf(node, "tag name is required")
		}
		t.Name = name
		return nil
	}

	if node.Kind != yaml.MappingNode {
		return nodeErrorf(node, "tag must be a string or mapping")
	}

	// Iterate over key-value pairs in the mapping
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		key := keyNode.Value

		if key == "name" {
			// Decode name field
			if err := valueNode.Decode(&t.Name); err != nil {
				return wrapNodeError(node, "failed to decode tag name", err)
			}
		} else {
			// All other fields are attributes
			var value interface{}
			if err := valueNode.Decode(&value); err != nil {
				return wrapNodeError(valueNode, fmt.Sprintf("failed to decode tag attribute %q", key), err)
			}
			t.Attributes[key] = value
		}
	}

	if t.Name == "" {
		return nodeErrorf(node, "tag name is required")
	}

	return nil
}

// ServiceDefaults holds default values for service configuration.
// Only shared, public, and autoconfigure fields are allowed in _default section.
type ServiceDefaults struct {
	Shared        *bool `yaml:"shared"`
	Public        *bool `yaml:"public"`
	Autoconfigure *bool `yaml:"autoconfigure"`
}

type RawService struct {
	Type               string          `yaml:"type"`
	Constructor        RawConstructor  `yaml:"constructor"`
	Shared             *bool           `yaml:"shared"`
	Public             *bool           `yaml:"public"`
	Autoconfigure      *bool           `yaml:"autoconfigure"`
	Decorates          string          `yaml:"decorates"`
	DecorationPriority int             `yaml:"decoration_priority"`
	Tags               []RawServiceTag `yaml:"tags"`
	Alias              string          `yaml:"alias"`

	// Node holds the service mapping node for location tracking
	Node *yaml.Node `yaml:"-"`
}

func (s *RawService) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		var ref string
		if err := node.Decode(&ref); err != nil {
			return err
		}
		*s = RawService{Alias: ref, Node: node}
		return nil
	case yaml.MappingNode:
		type alias RawService
		var decoded alias
		if err := node.Decode(&decoded); err != nil {
			return err
		}
		*s = RawService(decoded)
		s.Node = node
		return nil
	default:
		return nodeErrorf(node, "service must be a mapping or alias")
	}
}

type RawConstructor struct {
	Func   string        `yaml:"func"`
	Method string        `yaml:"method"`
	Args   []RawArgument `yaml:"args"`

	// Nodes for location tracking
	Node *yaml.Node `yaml:"-"`
}

func (c *RawConstructor) UnmarshalYAML(node *yaml.Node) error {
	// Preserve constructor node
	c.Node = node

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

	// Capture Func and Method node locations from the mapping
	if node.Kind == yaml.MappingNode {
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			switch keyNode.Value {
			case "func", "method":
				c.Node = node.Content[i+1]
			}
		}
	}

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
	// Always preserve the node for location tracking
	a.Node = node

	if node.Kind == yaml.ScalarNode && node.Tag == "!!str" {
		val := node.Value
		a.Value = &val
		return nil
	}

	return nil
}
