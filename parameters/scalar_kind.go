package parameters

import (
	"fmt"
)

// ScalarKind identifies a supported parameter target scalar. It is the
// shared vocabulary between generation-time validation and code generation:
// Cast dispatches to the matching Caster method, ResolverMethod names the
// matching Resolver method.
type ScalarKind int

const (
	ScalarString ScalarKind = iota
	ScalarBool

	ScalarInt
	ScalarInt8
	ScalarInt16
	ScalarInt32
	ScalarInt64

	ScalarUint
	ScalarUint8
	ScalarUint16
	ScalarUint32
	ScalarUint64

	ScalarFloat32
	ScalarFloat64

	ScalarDuration
	ScalarTime
)

// Cast converts value through the caster method matching the kind.
func (k ScalarKind) Cast(c Caster, value any) (any, error) {
	switch k {
	case ScalarString:
		return c.ToString(value)
	case ScalarBool:
		return c.ToBool(value)
	case ScalarInt:
		return c.ToInt(value)
	case ScalarInt8:
		return c.ToInt8(value)
	case ScalarInt16:
		return c.ToInt16(value)
	case ScalarInt32:
		return c.ToInt32(value)
	case ScalarInt64:
		return c.ToInt64(value)
	case ScalarUint:
		return c.ToUint(value)
	case ScalarUint8:
		return c.ToUint8(value)
	case ScalarUint16:
		return c.ToUint16(value)
	case ScalarUint32:
		return c.ToUint32(value)
	case ScalarUint64:
		return c.ToUint64(value)
	case ScalarFloat32:
		return c.ToFloat32(value)
	case ScalarFloat64:
		return c.ToFloat64(value)
	case ScalarDuration:
		return c.ToDuration(value)
	case ScalarTime:
		return c.ToTime(value)
	default:
		return nil, fmt.Errorf("unknown scalar kind %d", k)
	}
}

// ResolverMethod returns the Resolver method name for the kind. It returns
// an empty string for an unknown kind.
func (k ScalarKind) ResolverMethod() string {
	switch k {
	case ScalarString:
		return "String"
	case ScalarBool:
		return "Bool"
	case ScalarInt:
		return "Int"
	case ScalarInt8:
		return "Int8"
	case ScalarInt16:
		return "Int16"
	case ScalarInt32:
		return "Int32"
	case ScalarInt64:
		return "Int64"
	case ScalarUint:
		return "Uint"
	case ScalarUint8:
		return "Uint8"
	case ScalarUint16:
		return "Uint16"
	case ScalarUint32:
		return "Uint32"
	case ScalarUint64:
		return "Uint64"
	case ScalarFloat32:
		return "Float32"
	case ScalarFloat64:
		return "Float64"
	case ScalarDuration:
		return "Duration"
	case ScalarTime:
		return "Time"
	default:
		return ""
	}
}
