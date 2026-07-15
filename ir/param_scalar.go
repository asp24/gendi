package ir

import (
	"fmt"
	"go/types"

	"github.com/gendi-org/gendi/parameters"
	"github.com/gendi-org/gendi/typeres"
)

// ParamScalarKind classifies a parameter injection target. needsConversion
// reports whether generated code must wrap the converted value in a static
// conversion to a named target type. Exact time.Duration targets map to
// ScalarDuration; other named types with underlying int64 map to ScalarInt64
// followed by a conversion.
func ParamScalarKind(target types.Type) (kind parameters.ScalarKind, needsConversion bool, err error) {
	if typeres.IsDuration(target) {
		return parameters.ScalarDuration, false, nil
	}
	if typeres.IsTime(target) {
		return parameters.ScalarTime, false, nil
	}
	basic, ok := target.Underlying().(*types.Basic)
	if !ok {
		return 0, false, fmt.Errorf("parameter cannot be injected as %s: unsupported target type", target)
	}
	switch basic.Kind() {
	case types.String:
		kind = parameters.ScalarString
	case types.Bool:
		kind = parameters.ScalarBool
	case types.Int:
		kind = parameters.ScalarInt
	case types.Int8:
		kind = parameters.ScalarInt8
	case types.Int16:
		kind = parameters.ScalarInt16
	case types.Int32:
		kind = parameters.ScalarInt32
	case types.Int64:
		kind = parameters.ScalarInt64
	case types.Uint:
		kind = parameters.ScalarUint
	case types.Uint8:
		kind = parameters.ScalarUint8
	case types.Uint16:
		kind = parameters.ScalarUint16
	case types.Uint32:
		kind = parameters.ScalarUint32
	case types.Uint64:
		kind = parameters.ScalarUint64
	case types.Float32:
		kind = parameters.ScalarFloat32
	case types.Float64:
		kind = parameters.ScalarFloat64
	default:
		return 0, false, fmt.Errorf("parameter cannot be injected as %s: unsupported target type", target)
	}
	_, isNamed := types.Unalias(target).(*types.Named)
	return kind, isNamed, nil
}
