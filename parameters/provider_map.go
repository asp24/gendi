package parameters

import (
	"fmt"
)

// ProviderMap looks up parameters from a map. Stored values are returned
// as-is; the caster owns all conversion semantics.
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

func (p *ProviderMap) Lookup(name string) (any, error) {
	val, ok := p.values[name]
	if !ok {
		return nil, fmt.Errorf("parameter %q: %w", name, ErrParameterNotFound)
	}
	return val, nil
}
