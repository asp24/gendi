package parameters

import (
	"fmt"
	"strconv"
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
	switch cVal := val.(type) {
	case int8:
		return strconv.FormatInt(int64(cVal), 10), nil
	case int16:
		return strconv.FormatInt(int64(cVal), 10), nil
	case int32:
		return strconv.FormatInt(int64(cVal), 10), nil
	case int64:
		return strconv.FormatInt(cVal, 10), nil
	case int:
		return strconv.FormatInt(int64(cVal), 10), nil
	case uint:
		return strconv.FormatUint(uint64(cVal), 10), nil
	case uint8:
		return strconv.FormatUint(uint64(cVal), 10), nil
	case uint16:
		return strconv.FormatUint(uint64(cVal), 10), nil
	case uint32:
		return strconv.FormatUint(uint64(cVal), 10), nil
	case uint64:
		return strconv.FormatUint(cVal, 10), nil
	case float32:
		return strconv.FormatFloat(float64(cVal), 'f', -1, 32), nil
	case float64:
		return strconv.FormatFloat(cVal, 'f', -1, 64), nil
	case string:
		return cVal, nil
	default:
		return "", fmt.Errorf("parameter %q: expected string, got %T", name, val)
	}
}

func (p *ProviderMap) GetInt(name string) (int, error) {
	val, err := p.lookup(name)
	if err != nil {
		return 0, err
	}
	switch cVal := val.(type) {
	case int8:
		return int(cVal), nil
	case int16:
		return int(cVal), nil
	case int32:
		return int(cVal), nil
	case int64:
		return int(cVal), nil

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

func (p *ProviderMap) GetDuration(name string) (time.Duration, error) {
	val, err := p.lookup(name)
	if err != nil {
		return 0, err
	}
	switch cVal := val.(type) {
	case time.Duration:
		return cVal, nil
	case string:
		parsed, err := time.ParseDuration(cVal)
		if err != nil {
			return 0, fmt.Errorf("parameter %q: invalid duration: %w", name, err)
		}
		return parsed, nil
	case int:
		return time.Duration(cVal), nil
	case int8:
		return time.Duration(cVal), nil
	case int16:
		return time.Duration(cVal), nil
	case int32:
		return time.Duration(cVal), nil
	case int64:
		return time.Duration(cVal), nil
	case uint:
		return time.Duration(cVal), nil
	case uint8:
		return time.Duration(cVal), nil
	case uint16:
		return time.Duration(cVal), nil
	case uint32:
		return time.Duration(cVal), nil
	case uint64:
		if cVal > uint64(^uint64(0)>>1) {
			return 0, fmt.Errorf("parameter %q: duration overflows int64", name)
		}
		return time.Duration(cVal), nil
	default:
		return 0, fmt.Errorf("parameter %q: expected duration, got %T", name, val)
	}
}
