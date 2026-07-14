package yaml

import (
	"fmt"

	yamllib "github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
)

// RawConfig is the YAML-specific representation of a config file.
type RawConfig struct {
	Imports    []RawImport             `yaml:"imports"`
	Parameters map[string]RawParameter `yaml:"parameters"`
	Tags       map[string]RawTag       `yaml:"tags"`
	Services   map[string]*RawService  `yaml:"services"`
}

func (c *RawConfig) UnmarshalYAML(node ast.Node) error {
	type alias RawConfig
	var decoded alias
	if err := yamllib.NodeToValue(node, &decoded); err != nil {
		return err
	}

	if mapping, ok := node.(*ast.MappingNode); ok {
		for _, kv := range mapping.Values {
			switch keyString(kv.Key) {
			case "parameters":
				if pm, ok := kv.Value.(*ast.MappingNode); ok {
					for _, pv := range pm.Values {
						name := keyString(pv.Key)
						if param, ok := decoded.Parameters[name]; ok {
							param.Node = pv.Value
							decoded.Parameters[name] = param
						}
					}
				}
			case "tags":
				if tm, ok := kv.Value.(*ast.MappingNode); ok {
					for _, tv := range tm.Values {
						name := keyString(tv.Key)
						if tag, ok := decoded.Tags[name]; ok {
							tag.Node = tv.Value
							decoded.Tags[name] = tag
						}
					}
				}
			case "services":
				// goccy does not call RawService.UnmarshalYAML for null
				// values, leaving nil entries in the map.
				if sm, ok := kv.Value.(*ast.MappingNode); ok {
					for _, sv := range sm.Values {
						if decoded.Services[keyString(sv.Key)] == nil {
							node := sv.Value
							if node == nil {
								node = sv.Key
							}
							return nodeErrorf(node, "service must be a mapping or alias")
						}
					}
				}
			}
		}
	}

	*c = RawConfig(decoded)
	return nil
}

// keyString extracts a scalar string from a key node. Returns "" for
// any non-scalar key (which never occurs in well-formed configs).
func keyString(n ast.Node) string {
	if s, ok := n.(*ast.StringNode); ok {
		return s.Value
	}
	return ""
}

type RawImport struct {
	Path    string   `yaml:"path"`
	Exclude []string `yaml:"exclude"`
}

func (i *RawImport) UnmarshalYAML(node ast.Node) error {
	switch n := node.(type) {
	case *ast.StringNode:
		i.Path = n.Value
		return nil
	case *ast.LiteralNode:
		if n.Value != nil {
			i.Path = n.Value.Value
		}
		return nil
	case *ast.MappingNode:
		type alias RawImport
		var decoded alias
		if err := yamllib.NodeToValue(node, &decoded); err != nil {
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
	Type  string   `yaml:"type"`
	Value ast.Node `yaml:"value"`

	// Node holds the full parameter mapping node for location tracking.
	Node ast.Node `yaml:"-"`
}

type RawTag struct {
	ElementType   string `yaml:"element_type"`
	SortBy        string `yaml:"sort_by"`
	Public        bool   `yaml:"public"`
	Autoconfigure bool   `yaml:"autoconfigure"`

	// Node holds the tag mapping node for location tracking.
	Node ast.Node `yaml:"-"`
}

type RawServiceTag struct {
	Name       string
	Attributes map[string]interface{}

	// Node holds the tag mapping node for location tracking.
	Node ast.Node `yaml:"-"`
}

func (t *RawServiceTag) UnmarshalYAML(node ast.Node) error {
	t.Node = node
	t.Attributes = make(map[string]interface{})

	switch n := node.(type) {
	case *ast.StringNode:
		if n.Value == "" {
			return nodeErrorf(node, "tag name is required")
		}
		t.Name = n.Value
		return nil
	case *ast.LiteralNode:
		if n.Value == nil || n.Value.Value == "" {
			return nodeErrorf(node, "tag name is required")
		}
		t.Name = n.Value.Value
		return nil
	case *ast.MappingNode:
		for _, kv := range n.Values {
			key := keyString(kv.Key)
			if key == "name" {
				if err := yamllib.NodeToValue(kv.Value, &t.Name); err != nil {
					return wrapNodeError(node, "failed to decode tag name", err)
				}
			} else {
				var value interface{}
				if err := yamllib.NodeToValue(kv.Value, &value); err != nil {
					return wrapNodeError(kv.Value, fmt.Sprintf("failed to decode tag attribute %q", key), err)
				}
				t.Attributes[key] = value
			}
		}
		if t.Name == "" {
			return nodeErrorf(node, "tag name is required")
		}
		return nil
	default:
		return nodeErrorf(node, "tag must be a string or mapping")
	}
}

// ServiceDefaults holds default values for service configuration.
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

	// Node holds the service mapping node for location tracking.
	Node ast.Node `yaml:"-"`
}

func (s *RawService) UnmarshalYAML(node ast.Node) error {
	switch n := node.(type) {
	case *ast.StringNode:
		*s = RawService{Alias: n.Value, Node: node}
		return nil
	case *ast.LiteralNode:
		ref := ""
		if n.Value != nil {
			ref = n.Value.Value
		}
		*s = RawService{Alias: ref, Node: node}
		return nil
	case *ast.MappingNode:
		type alias RawService
		var decoded alias
		if err := yamllib.NodeToValue(node, &decoded); err != nil {
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

	// Node points at func/method scalar for location tracking.
	Node ast.Node `yaml:"-"`
}

func (c *RawConstructor) UnmarshalYAML(node ast.Node) error {
	c.Node = node

	type raw struct {
		Func   string `yaml:"func"`
		Method string `yaml:"method"`
	}
	var decoded raw
	if err := yamllib.NodeToValue(node, &decoded); err != nil {
		return err
	}
	c.Func = decoded.Func
	c.Method = decoded.Method

	// Walk the AST manually to grab arg nodes. Cannot use
	// NodeToValue with `Args []ast.Node` because goccy v1.19.2 decodes
	// null sequence entries as typed-nil interfaces, losing the
	// *ast.NullNode information needed by convertLiteral.
	var argNodes []ast.Node
	if mapping, ok := node.(*ast.MappingNode); ok {
		for _, kv := range mapping.Values {
			switch keyString(kv.Key) {
			case "func", "method":
				c.Node = kv.Value
			case "args":
				if seq, ok := kv.Value.(*ast.SequenceNode); ok {
					argNodes = seq.Values
				}
			}
		}
	}

	if len(argNodes) == 0 {
		return nil
	}
	c.Args = make([]RawArgument, len(argNodes))
	for i := range argNodes {
		if err := c.Args[i].UnmarshalYAML(argNodes[i]); err != nil {
			return err
		}
	}
	return nil
}

type RawArgument struct {
	Value *string
	Node  ast.Node
}

func (a *RawArgument) UnmarshalYAML(node ast.Node) error {
	a.Node = node

	switch n := node.(type) {
	case *ast.StringNode:
		val := n.Value
		a.Value = &val
	case *ast.LiteralNode:
		if n.Value != nil {
			val := n.Value.Value
			a.Value = &val
		}
	}
	return nil
}
