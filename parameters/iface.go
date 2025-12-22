package parameters

import (
	"errors"
)

// ErrParameterNotFound is returned when a parameter is missing.
var ErrParameterNotFound = errors.New("parameter not found")

// Provider supplies typed parameters by name.
type Provider interface {
	Has(name string) bool
	GetString(name string) (string, error)
	GetInt(name string) (int, error)
	GetBool(name string) (bool, error)
	GetFloat(name string) (float64, error)
}
