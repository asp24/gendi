package parameters

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

// ProviderStructTag reads parameter values from struct fields tagged with `di-param`.
type ProviderStructTag struct {
	value any
}

// NewProviderStructTag returns a struct-tag-backed provider.
func NewProviderStructTag(value any) *ProviderStructTag {
	return &ProviderStructTag{value: value}
}

func (p *ProviderStructTag) Lookup(name string) (any, error) {
	val, err := p.lookup(name)
	if err != nil {
		return nil, fmt.Errorf("parameter %q: %w", name, err)
	}
	return normalizeValue(val), nil
}

// normalizeValue reduces a reflectively read value to the base-kind scalar
// form of the Lookup contract: named types collapse to their underlying
// kind, exact time.Time and time.Duration are preserved, and non-scalar
// values pass through for the caster to reject with a typed error.
func normalizeValue(v reflect.Value) any {
	switch v.Type() {
	case reflect.TypeFor[time.Duration](), reflect.TypeFor[time.Time]():
		return v.Interface()
	}
	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Bool:
		return v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int()
	// Uintptr is deliberately absent: it is not a supported scalar, so it
	// must reach the caster as-is and be rejected uniformly across providers.
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint()
	case reflect.Float32:
		return float32(v.Float())
	case reflect.Float64:
		return v.Float()
	default:
		return v.Interface()
	}
}

func (p *ProviderStructTag) lookup(name string) (reflect.Value, error) {
	if p == nil || p.value == nil {
		return reflect.Value{}, ErrParameterNotFound
	}
	root := reflect.ValueOf(p.value)
	root, ok := derefValue(root)
	if !ok || root.Kind() != reflect.Struct {
		return reflect.Value{}, ErrParameterNotFound
	}
	if val, ok := lookupStruct(root, name); ok {
		return val, nil
	}
	return reflect.Value{}, ErrParameterNotFound
}

func lookupStruct(v reflect.Value, name string) (reflect.Value, bool) {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}
		tag := strings.TrimSpace(strings.Split(field.Tag.Get("di-param"), ",")[0])
		if tag == "-" {
			continue
		}
		fieldVal := v.Field(i)
		fieldVal, ok := derefValue(fieldVal)
		if !ok {
			continue
		}

		if tag != "" {
			switch fieldVal.Kind() {
			case reflect.Map:
				if after, ok0 := strings.CutPrefix(name, tag+"."); ok0 {
					keyPath := after
					if val, ok := lookupMap(fieldVal, keyPath); ok {
						return val, true
					}
				}
			case reflect.Struct:
				// Exact match returns the struct itself as a leaf value
				// (e.g. a time.Time field); the caster rejects other structs.
				if name == tag {
					return fieldVal, true
				}
				if after, ok0 := strings.CutPrefix(name, tag+"."); ok0 {
					nestedName := after
					if val, ok := lookupStruct(fieldVal, nestedName); ok {
						return val, true
					}
				}
			default:
				if name == tag {
					return fieldVal, true
				}
			}
			continue
		}

		if fieldVal.Kind() == reflect.Struct {
			if val, ok := lookupStruct(fieldVal, name); ok {
				return val, true
			}
		}
	}
	return reflect.Value{}, false
}

func lookupMap(m reflect.Value, keyPath string) (reflect.Value, bool) {
	parts := strings.Split(keyPath, ".")
	current := m
	for i, part := range parts {
		if current.Kind() != reflect.Map || current.Type().Key().Kind() != reflect.String {
			return reflect.Value{}, false
		}
		// Convert handles named key types (e.g. type Key string).
		val := current.MapIndex(reflect.ValueOf(part).Convert(current.Type().Key()))
		if !val.IsValid() {
			return reflect.Value{}, false
		}
		val, ok := derefValue(val)
		if !ok {
			return reflect.Value{}, false
		}
		if i == len(parts)-1 {
			return val, true
		}
		current = val
	}
	return reflect.Value{}, false
}

func derefValue(v reflect.Value) (reflect.Value, bool) {
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return reflect.Value{}, false
		}
		v = v.Elem()
	}
	return v, true
}
