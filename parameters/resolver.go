package parameters

import (
	"time"
)

// Resolver combines a Provider (lookup) with a Caster (conversion) behind
// one call per injection site. It is a deliberately dumb facade: lookup
// semantics stay in Provider, conversion policy stays in Caster, and both
// remain the extension points. Resolver is a concrete type, not an
// interface, so the two responsibilities cannot be fused back together by
// an alternative implementation. It is immutable after construction; use
// WithCaster to derive a resolver with a different conversion policy.
type Resolver struct {
	provider Provider
	caster   Caster
}

// NewResolver returns a Resolver over the given provider and caster. Both
// arguments are stored as-is: a misconfigured (nil) dependency fails loudly
// on first use instead of being silently replaced by a default.
func NewResolver(p Provider, c Caster) *Resolver {
	return &Resolver{provider: p, caster: c}
}

// WithCaster returns a copy of the resolver that converts through the given
// caster. The receiver is left unchanged and the provider is shared.
func (r *Resolver) WithCaster(c Caster) *Resolver {
	return &Resolver{provider: r.provider, caster: c}
}

func (r *Resolver) String(name string) (string, error) {
	raw, err := r.provider.Lookup(name)
	if err != nil {
		return "", err
	}
	return r.caster.ToString(raw)
}

func (r *Resolver) Bool(name string) (bool, error) {
	raw, err := r.provider.Lookup(name)
	if err != nil {
		return false, err
	}
	return r.caster.ToBool(raw)
}

func (r *Resolver) Int(name string) (int, error) {
	raw, err := r.provider.Lookup(name)
	if err != nil {
		return 0, err
	}
	return r.caster.ToInt(raw)
}

func (r *Resolver) Int8(name string) (int8, error) {
	raw, err := r.provider.Lookup(name)
	if err != nil {
		return 0, err
	}
	return r.caster.ToInt8(raw)
}

func (r *Resolver) Int16(name string) (int16, error) {
	raw, err := r.provider.Lookup(name)
	if err != nil {
		return 0, err
	}
	return r.caster.ToInt16(raw)
}

func (r *Resolver) Int32(name string) (int32, error) {
	raw, err := r.provider.Lookup(name)
	if err != nil {
		return 0, err
	}
	return r.caster.ToInt32(raw)
}

func (r *Resolver) Int64(name string) (int64, error) {
	raw, err := r.provider.Lookup(name)
	if err != nil {
		return 0, err
	}
	return r.caster.ToInt64(raw)
}

func (r *Resolver) Uint(name string) (uint, error) {
	raw, err := r.provider.Lookup(name)
	if err != nil {
		return 0, err
	}
	return r.caster.ToUint(raw)
}

func (r *Resolver) Uint8(name string) (uint8, error) {
	raw, err := r.provider.Lookup(name)
	if err != nil {
		return 0, err
	}
	return r.caster.ToUint8(raw)
}

func (r *Resolver) Uint16(name string) (uint16, error) {
	raw, err := r.provider.Lookup(name)
	if err != nil {
		return 0, err
	}
	return r.caster.ToUint16(raw)
}

func (r *Resolver) Uint32(name string) (uint32, error) {
	raw, err := r.provider.Lookup(name)
	if err != nil {
		return 0, err
	}
	return r.caster.ToUint32(raw)
}

func (r *Resolver) Uint64(name string) (uint64, error) {
	raw, err := r.provider.Lookup(name)
	if err != nil {
		return 0, err
	}
	return r.caster.ToUint64(raw)
}

func (r *Resolver) Float32(name string) (float32, error) {
	raw, err := r.provider.Lookup(name)
	if err != nil {
		return 0, err
	}
	return r.caster.ToFloat32(raw)
}

func (r *Resolver) Float64(name string) (float64, error) {
	raw, err := r.provider.Lookup(name)
	if err != nil {
		return 0, err
	}
	return r.caster.ToFloat64(raw)
}

func (r *Resolver) Duration(name string) (time.Duration, error) {
	raw, err := r.provider.Lookup(name)
	if err != nil {
		return 0, err
	}
	return r.caster.ToDuration(raw)
}

func (r *Resolver) Time(name string) (time.Time, error) {
	raw, err := r.provider.Lookup(name)
	if err != nil {
		return time.Time{}, err
	}
	return r.caster.ToTime(raw)
}
