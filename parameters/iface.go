package parameters

import (
	"errors"
)

// ErrParameterNotFound is returned when a parameter is missing.
var ErrParameterNotFound = errors.New("parameter not found")

// Provider locates a raw parameter value by name; all conversion semantics
// live in Caster.
//
// Lookup returns values of base scalar kinds — string, bool, signed and
// unsigned integers, floats — plus exact time.Time and time.Duration.
// Providers that read values reflectively normalize named types to their
// base kinds; in-memory providers may return stored values as-is, in which
// case the caster rejects unsupported values with an error naming the type.
// A missing parameter is reported with an error wrapping ErrParameterNotFound.
type Provider interface {
	Lookup(name string) (any, error)
}
