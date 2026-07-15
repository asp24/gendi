package ir

import (
	"fmt"
	"go/types"

	"github.com/asp24/gendi/typeres"
)

// CasterMethod returns the parameters.Caster method used to convert a raw
// parameter value to the target type, and whether generated code must wrap
// the result in a static conversion to a named target type. Exact
// time.Duration targets use ToDuration; other named types with underlying
// int64 use ToInt64 followed by a conversion.
func CasterMethod(target types.Type) (method string, needsConversion bool, err error) {
	if typeres.IsDuration(target) {
		return "ToDuration", false, nil
	}
	if typeres.IsTime(target) {
		return "ToTime", false, nil
	}
	basic, ok := target.Underlying().(*types.Basic)
	if !ok {
		return "", false, fmt.Errorf("parameter cannot be injected as %s: unsupported target type", target)
	}
	switch basic.Kind() {
	case types.String:
		method = "ToString"
	case types.Bool:
		method = "ToBool"
	case types.Int:
		method = "ToInt"
	case types.Int8:
		method = "ToInt8"
	case types.Int16:
		method = "ToInt16"
	case types.Int32:
		method = "ToInt32"
	case types.Int64:
		method = "ToInt64"
	case types.Uint:
		method = "ToUint"
	case types.Uint8:
		method = "ToUint8"
	case types.Uint16:
		method = "ToUint16"
	case types.Uint32:
		method = "ToUint32"
	case types.Uint64:
		method = "ToUint64"
	case types.Float32:
		method = "ToFloat32"
	case types.Float64:
		method = "ToFloat64"
	default:
		return "", false, fmt.Errorf("parameter cannot be injected as %s: unsupported target type", target)
	}
	_, isNamed := types.Unalias(target).(*types.Named)
	return method, isNamed, nil
}
