// Package yaml provides YAML parsing for DI configuration files.
package yaml

import (
	"fmt"
	"strings"

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
	return p.convertConfigWithDir(raw, "")
}

func (p *Parser) convertConfigWithDir(raw *RawConfig, configDir string) (*di.Config, error) {
	cfg := &di.Config{
		Parameters: make(map[string]di.Parameter),
		Tags:       make(map[string]di.Tag),
		Services:   make(map[string]di.Service),
	}

	// Resolve $this package path from config file directory
	var thisPackage string
	if configDir != "" {
		pkg, err := resolvePackagePath(configDir)
		if err != nil {
			// If resolution fails, $this will remain empty and cause an error if used
			thisPackage = ""
		} else {
			thisPackage = pkg
		}
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
			Public:      tag.Public,
		}
	}

	// Extract and parse _default if present
	var defaults *ServiceDefaults
	if defaultSvc, ok := raw.Services["_default"]; ok {
		defaults = &ServiceDefaults{
			Shared: defaultSvc.Shared,
			Public: defaultSvc.Public,
		}
		// Validate that _default only contains allowed fields
		if err := p.validateDefaults(defaultSvc); err != nil {
			return nil, fmt.Errorf("_default: %w", err)
		}
	}

	// Convert services
	for name, svc := range raw.Services {
		if name == "_default" {
			continue // Skip _default itself
		}
		converted, err := p.convertServiceWithPackage(svc, defaults, thisPackage)
		if err != nil {
			return nil, fmt.Errorf("service %q: %w", name, err)
		}
		cfg.Services[name] = converted
	}

	return cfg, nil
}

func (p *Parser) convertService(raw *RawService, defaults *ServiceDefaults) (di.Service, error) {
	return p.convertServiceWithPackage(raw, defaults, "")
}

func (p *Parser) convertServiceWithPackage(raw *RawService, defaults *ServiceDefaults, thisPackage string) (di.Service, error) {
	// Apply defaults if not explicitly set
	defaultShared := true
	if defaults != nil && defaults.Shared != nil {
		defaultShared = *defaults.Shared
	}

	shared := defaultShared
	if raw.Shared != nil {
		shared = *raw.Shared
	}

	public := raw.Public
	if public == nil && defaults != nil && defaults.Public != nil {
		public = defaults.Public
	}

	// Convert to non-pointer for di.Service
	publicBool := false
	if public != nil {
		publicBool = *public
	}

	svc := di.Service{
		Type:               raw.Type,
		Shared:             shared,
		Public:             publicBool,
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

	// Substitute $this with the resolved package path
	if thisPackage != "" {
		// Substitute in type field (can appear anywhere due to type prefixes like *, [], etc.)
		if strings.Contains(svc.Type, "$this.") {
			svc.Type = strings.ReplaceAll(svc.Type, "$this.", thisPackage+".")
		}
		// Substitute in constructor fields (must be at start)
		if strings.HasPrefix(svc.Constructor.Func, "$this.") {
			svc.Constructor.Func = strings.Replace(svc.Constructor.Func, "$this.", thisPackage+".", 1)
		}
		if strings.HasPrefix(svc.Constructor.Method, "$this.") {
			svc.Constructor.Method = strings.Replace(svc.Constructor.Method, "$this.", thisPackage+".", 1)
		}
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

// validateDefaults ensures _default only contains allowed fields (shared, public)
func (p *Parser) validateDefaults(raw *RawService) error {
	if raw.Type != "" {
		return fmt.Errorf("field 'type' is not allowed in _default")
	}
	if raw.Constructor.Func != "" || raw.Constructor.Method != "" || len(raw.Constructor.Args) > 0 {
		return fmt.Errorf("field 'constructor' is not allowed in _default")
	}
	if raw.Alias != "" {
		return fmt.Errorf("field 'alias' is not allowed in _default")
	}
	if raw.Decorates != "" {
		return fmt.Errorf("field 'decorates' is not allowed in _default")
	}
	if raw.DecorationPriority != 0 {
		return fmt.Errorf("field 'decoration_priority' is not allowed in _default")
	}
	if len(raw.Tags) > 0 {
		return fmt.Errorf("field 'tags' is not allowed in _default")
	}
	// Only shared and public are allowed, which we already extracted
	return nil
}
