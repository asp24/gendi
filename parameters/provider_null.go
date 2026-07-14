package parameters

import (
	"fmt"
	"time"
)

type ProviderNull struct {
}

var ProviderNullInstance = &ProviderNull{}

func NewProviderNull() *ProviderNull {
	return ProviderNullInstance
}

func (p *ProviderNull) Has(_ string) bool {
	return false
}

func (p *ProviderNull) GetString(name string) (string, error) {
	return "", fmt.Errorf("parameter %q: %w", name, ErrParameterNotFound)
}

func (p *ProviderNull) GetInt(name string) (int, error) {
	return 0, fmt.Errorf("parameter %q: %w", name, ErrParameterNotFound)
}

func (p *ProviderNull) GetBool(name string) (bool, error) {
	return false, fmt.Errorf("parameter %q: %w", name, ErrParameterNotFound)
}

func (p *ProviderNull) GetFloat(name string) (float64, error) {
	return 0.0, fmt.Errorf("parameter %q: %w", name, ErrParameterNotFound)
}

func (p *ProviderNull) GetDuration(name string) (time.Duration, error) {
	return 0, fmt.Errorf("parameter %q: %w", name, ErrParameterNotFound)
}
