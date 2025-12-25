package parameters

import (
	"fmt"
	"time"
)

// ProviderMap looks up parameters from a map.
type ProviderMap struct {
	values map[string]any
}

// NewProviderMap returns a map-backed Provider.
func NewProviderMap(values map[string]any) *ProviderMap {
	if values == nil {
		values = make(map[string]any)
	}

	return &ProviderMap{values: values}
}

func (p *ProviderMap) Has(name string) bool {
	_, ok := p.values[name]

	return ok
}

func (p *ProviderMap) lookup(name string) (any, error) {
	val, ok := p.values[name]
	if !ok {
		return nil, fmt.Errorf("parameter %q: %w", name, ErrParameterNotFound)
	}
	return val, nil
}

func (p *ProviderMap) GetString(name string) (string, error) {
	val, err := p.lookup(name)
	if err != nil {
		return "", err
	}
	result, err := convertToString(val)
	if err != nil {
		return "", fmt.Errorf("parameter %q: %w", name, err)
	}
	return result, nil
}

func (p *ProviderMap) GetInt(name string) (int, error) {
	val, err := p.lookup(name)
	if err != nil {
		return 0, err
	}
	result, err := convertToInt(val)
	if err != nil {
		return 0, fmt.Errorf("parameter %q: %w", name, err)
	}
	return result, nil
}

func (p *ProviderMap) GetBool(name string) (bool, error) {
	val, err := p.lookup(name)
	if err != nil {
		return false, err
	}
	result, err := convertToBool(val)
	if err != nil {
		return false, fmt.Errorf("parameter %q: %w", name, err)
	}
	return result, nil
}

func (p *ProviderMap) GetFloat(name string) (float64, error) {
	val, err := p.lookup(name)
	if err != nil {
		return 0, err
	}
	result, err := convertToFloat(val)
	if err != nil {
		return 0, fmt.Errorf("parameter %q: %w", name, err)
	}
	return result, nil
}

func (p *ProviderMap) GetDuration(name string) (time.Duration, error) {
	val, err := p.lookup(name)
	if err != nil {
		return 0, err
	}
	result, err := convertToDuration(val)
	if err != nil {
		return 0, fmt.Errorf("parameter %q: %w", name, err)
	}
	return result, nil
}
