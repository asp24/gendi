package parameters

import (
	"fmt"
)

type ProviderNull struct {
}

var ProviderNullInstance = &ProviderNull{}

func NewProviderNull() *ProviderNull {
	return ProviderNullInstance
}

func (p *ProviderNull) Lookup(name string) (any, error) {
	return nil, fmt.Errorf("parameter %q: %w", name, ErrParameterNotFound)
}
