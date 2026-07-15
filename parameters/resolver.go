package parameters

import (
	"time"
)

// Resolver combines a Provider (lookup) with a Caster (conversion) behind
// one call per injection site. It is a deliberately dumb facade: lookup
// semantics stay in Provider, conversion policy stays in Caster, and both
// remain the extension points. Resolver is a concrete type, not an
// interface, so the two responsibilities cannot be fused back together by
// an alternative implementation.
type Resolver struct {
	Provider Provider
	Caster   Caster
}

// NewResolver returns a Resolver over the given provider and caster.
// A nil provider falls back to ProviderNullInstance and a nil caster to
// StandardCaster.
func NewResolver(p Provider, c Caster) *Resolver {
	if p == nil {
		p = ProviderNullInstance
	}
	if c == nil {
		c = StandardCaster{}
	}
	return &Resolver{Provider: p, Caster: c}
}

func (r Resolver) String(name string) (string, error) {
	raw, err := r.Provider.Lookup(name)
	if err != nil {
		return "", err
	}
	return r.Caster.ToString(raw)
}

func (r Resolver) Bool(name string) (bool, error) {
	raw, err := r.Provider.Lookup(name)
	if err != nil {
		return false, err
	}
	return r.Caster.ToBool(raw)
}

func (r Resolver) Int(name string) (int, error) {
	raw, err := r.Provider.Lookup(name)
	if err != nil {
		return 0, err
	}
	return r.Caster.ToInt(raw)
}

func (r Resolver) Int8(name string) (int8, error) {
	raw, err := r.Provider.Lookup(name)
	if err != nil {
		return 0, err
	}
	return r.Caster.ToInt8(raw)
}

func (r Resolver) Int16(name string) (int16, error) {
	raw, err := r.Provider.Lookup(name)
	if err != nil {
		return 0, err
	}
	return r.Caster.ToInt16(raw)
}

func (r Resolver) Int32(name string) (int32, error) {
	raw, err := r.Provider.Lookup(name)
	if err != nil {
		return 0, err
	}
	return r.Caster.ToInt32(raw)
}

func (r Resolver) Int64(name string) (int64, error) {
	raw, err := r.Provider.Lookup(name)
	if err != nil {
		return 0, err
	}
	return r.Caster.ToInt64(raw)
}

func (r Resolver) Uint(name string) (uint, error) {
	raw, err := r.Provider.Lookup(name)
	if err != nil {
		return 0, err
	}
	return r.Caster.ToUint(raw)
}

func (r Resolver) Uint8(name string) (uint8, error) {
	raw, err := r.Provider.Lookup(name)
	if err != nil {
		return 0, err
	}
	return r.Caster.ToUint8(raw)
}

func (r Resolver) Uint16(name string) (uint16, error) {
	raw, err := r.Provider.Lookup(name)
	if err != nil {
		return 0, err
	}
	return r.Caster.ToUint16(raw)
}

func (r Resolver) Uint32(name string) (uint32, error) {
	raw, err := r.Provider.Lookup(name)
	if err != nil {
		return 0, err
	}
	return r.Caster.ToUint32(raw)
}

func (r Resolver) Uint64(name string) (uint64, error) {
	raw, err := r.Provider.Lookup(name)
	if err != nil {
		return 0, err
	}
	return r.Caster.ToUint64(raw)
}

func (r Resolver) Float32(name string) (float32, error) {
	raw, err := r.Provider.Lookup(name)
	if err != nil {
		return 0, err
	}
	return r.Caster.ToFloat32(raw)
}

func (r Resolver) Float64(name string) (float64, error) {
	raw, err := r.Provider.Lookup(name)
	if err != nil {
		return 0, err
	}
	return r.Caster.ToFloat64(raw)
}

func (r Resolver) Duration(name string) (time.Duration, error) {
	raw, err := r.Provider.Lookup(name)
	if err != nil {
		return 0, err
	}
	return r.Caster.ToDuration(raw)
}

func (r Resolver) Time(name string) (time.Time, error) {
	raw, err := r.Provider.Lookup(name)
	if err != nil {
		return time.Time{}, err
	}
	return r.Caster.ToTime(raw)
}
