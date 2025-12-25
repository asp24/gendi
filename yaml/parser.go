// Package yaml provides YAML parsing for DI configuration files.
package yaml

import (
	"fmt"

	"gopkg.in/yaml.v3"

	di "github.com/asp24/gendi"
)

// Parser converts raw YAML structures to di.Config.
type Parser struct{}

// NewParser creates a new YAML parser.
func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) convertConfig(raw *RawConfig) (*di.Config, error) {
	cfg := &di.Config{
		Parameters: make(map[string]di.Parameter),
		Tags:       make(map[string]di.Tag),
		Services:   make(map[string]di.Service),
	}

	// Convert parameters
	for name, param := range raw.Parameters {
		lit, err := p.convertLiteral(&param.Value)
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
		converted, err := p.convertService(svc)
		if err != nil {
			return nil, fmt.Errorf("service %q: %w", name, err)
		}
		cfg.Services[name] = converted
	}

	return cfg, nil
}

func (p *Parser) convertService(raw *RawService) (di.Service, error) {
	svc := di.Service{
		Type:               raw.Type,
		Shared:             raw.Shared,
		Public:             raw.Public,
		Decorates:          raw.Decorates,
		DecorationPriority: raw.DecorationPriority,
	}

	if raw.Alias != "" {
		if IsServiceAlias(raw.Alias) {
			svc.Alias = ParseServiceAlias(raw.Alias)
		} else {
			svc.Alias = raw.Alias
		}
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
			converted, err := p.convertArgument(&arg)
			if err != nil {
				return di.Service{}, fmt.Errorf("arg[%d]: %w", i, err)
			}
			svc.Constructor.Args[i] = converted
		}
	}

	return svc, nil
}

func (p *Parser) convertArgument(raw *RawArgument) (di.Argument, error) {
	if raw.Value != nil {
		kind, val := ParseArgumentString(*raw.Value)
		if kind != di.ArgLiteral {
			return di.Argument{
				Kind:  kind,
				Value: val,
			}, nil
		}
		return di.Argument{
			Kind:    di.ArgLiteral,
			Literal: di.NewStringLiteral(*raw.Value),
		}, nil
	}

	if raw.Node != nil {
		lit, err := p.convertLiteral(raw.Node)
		if err != nil {
			return di.Argument{}, err
		}
		return di.Argument{
			Kind:    di.ArgLiteral,
			Literal: lit,
		}, nil
	}

	return di.Argument{}, fmt.Errorf("argument must have a value")
}

func (p *Parser) convertLiteral(node *yaml.Node) (di.Literal, error) {
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
