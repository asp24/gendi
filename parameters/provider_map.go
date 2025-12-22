package parameters

import (
	"fmt"
)

// ProviderMap looks up parameters from a map.
type ProviderMap struct {
	values map[string]interface{}
}

// NewProviderMap returns a map-backed Provider.
func NewProviderMap(values map[string]interface{}) *ProviderMap {
	return &ProviderMap{values: values}
}

func (p *ProviderMap) Has(name string) bool {
	_, ok := p.values[name]

	return ok
}

func (p *ProviderMap) GetString(name string) (string, error) {
	val, err := p.lookup(name)
	if err != nil {
		return "", err
	}
	s, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("parameter %q: expected string, got %T", name, val)
	}
	return s, nil
}

func (p *ProviderMap) GetInt(name string) (int, error) {
	val, err := p.lookup(name)
	if err != nil {
		return 0, err
	}
	i, ok := val.(int)
	if !ok {
		return 0, fmt.Errorf("parameter %q: expected int, got %T", name, val)
	}
	return i, nil
}

func (p *ProviderMap) GetBool(name string) (bool, error) {
	val, err := p.lookup(name)
	if err != nil {
		return false, err
	}
	b, ok := val.(bool)
	if !ok {
		return false, fmt.Errorf("parameter %q: expected bool, got %T", name, val)
	}
	return b, nil
}

func (p *ProviderMap) GetFloat(name string) (float64, error) {
	val, err := p.lookup(name)
	if err != nil {
		return 0, err
	}
	f, ok := val.(float64)
	if !ok {
		return 0, fmt.Errorf("parameter %q: expected float64, got %T", name, val)
	}
	return f, nil
}

func (p *ProviderMap) lookup(name string) (interface{}, error) {
	if p == nil || p.values == nil {
		return nil, fmt.Errorf("parameter %q: %w", name, ErrParameterNotFound)
	}
	val, ok := p.values[name]
	if !ok {
		return nil, fmt.Errorf("parameter %q: %w", name, ErrParameterNotFound)
	}
	return val, nil
}
