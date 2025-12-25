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

func (p *ProviderStructTag) Has(name string) bool {
	_, err := p.lookup(name)
	return err == nil
}

func (p *ProviderStructTag) GetString(name string) (string, error) {
	val, err := p.lookup(name)
	if err != nil {
		return "", err
	}
	if val.Kind() == reflect.String {
		return val.String(), nil
	}
	return "", fmt.Errorf("parameter %q: expected string, got %s", name, val.Type())
}

func (p *ProviderStructTag) GetInt(name string) (int, error) {
	val, err := p.lookup(name)
	if err != nil {
		return 0, err
	}
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v := val.Int()
		if v < int64(intMin) || v > int64(intMax) {
			return 0, fmt.Errorf("parameter %q: value out of int range", name)
		}
		return int(v), nil
	default:
		return 0, fmt.Errorf("parameter %q: expected int, got %s", name, val.Type())
	}
}

func (p *ProviderStructTag) GetBool(name string) (bool, error) {
	val, err := p.lookup(name)
	if err != nil {
		return false, err
	}
	if val.Kind() == reflect.Bool {
		return val.Bool(), nil
	}
	return false, fmt.Errorf("parameter %q: expected bool, got %s", name, val.Type())
}

func (p *ProviderStructTag) GetFloat(name string) (float64, error) {
	val, err := p.lookup(name)
	if err != nil {
		return 0, err
	}
	switch val.Kind() {
	case reflect.Float32, reflect.Float64:
		return val.Convert(reflect.TypeOf(float64(0))).Float(), nil
	default:
		return 0, fmt.Errorf("parameter %q: expected float64, got %s", name, val.Type())
	}
}

func (p *ProviderStructTag) GetDuration(name string) (time.Duration, error) {
	val, err := p.lookup(name)
	if err != nil {
		return 0, err
	}
	durationType := reflect.TypeOf(time.Duration(0))
	if val.Type() == durationType {
		return val.Convert(durationType).Interface().(time.Duration), nil
	}
	switch val.Kind() {
	case reflect.String:
		parsed, err := time.ParseDuration(val.String())
		if err != nil {
			return 0, fmt.Errorf("parameter %q: invalid duration: %w", name, err)
		}
		return parsed, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return time.Duration(val.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u := val.Uint()
		if u > uint64(^uint64(0)>>1) {
			return 0, fmt.Errorf("parameter %q: duration overflows int64", name)
		}
		return time.Duration(u), nil
	default:
		return 0, fmt.Errorf("parameter %q: expected duration, got %s", name, val.Type())
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
				if strings.HasPrefix(name, tag+".") {
					keyPath := strings.TrimPrefix(name, tag+".")
					if val, ok := lookupMap(fieldVal, keyPath); ok {
						return val, true
					}
				}
			case reflect.Struct:
				if strings.HasPrefix(name, tag+".") {
					nestedName := strings.TrimPrefix(name, tag+".")
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
	if m.Kind() != reflect.Map || m.Type().Key().Kind() != reflect.String {
		return reflect.Value{}, false
	}
	parts := strings.Split(keyPath, ".")
	current := m
	for i, part := range parts {
		val := current.MapIndex(reflect.ValueOf(part))
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
		if val.Kind() != reflect.Map {
			return reflect.Value{}, false
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

const (
	intMax = int(^uint(0) >> 1)
	intMin = -intMax - 1
)
