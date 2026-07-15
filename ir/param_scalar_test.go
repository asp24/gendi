package ir

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/gendi-org/gendi/parameters"
)

func newNamedType(pkgPath, pkgName, typeName string, underlying types.Type) types.Type {
	pkg := types.NewPackage(pkgPath, pkgName)
	obj := types.NewTypeName(token.NoPos, pkg, typeName, nil)
	return types.NewNamed(obj, underlying, nil)
}

func TestParamScalarKind(t *testing.T) {
	duration := newNamedType("time", "time", "Duration", types.Typ[types.Int64])
	timeTime := newNamedType("time", "time", "Time", types.NewStruct(nil, nil))
	namedPort := newNamedType("example.com/app", "app", "Port", types.Typ[types.Int])
	// A named type defined as `type Timeout time.Duration` has underlying
	// int64, so it routes through ScalarInt64 plus a static conversion.
	namedTimeout := newNamedType("example.com/app", "app", "Timeout", types.Typ[types.Int64])

	tests := []struct {
		name     string
		target   types.Type
		kind     parameters.ScalarKind
		needConv bool
	}{
		{"string", types.Typ[types.String], parameters.ScalarString, false},
		{"bool", types.Typ[types.Bool], parameters.ScalarBool, false},
		{"int", types.Typ[types.Int], parameters.ScalarInt, false},
		{"int8", types.Typ[types.Int8], parameters.ScalarInt8, false},
		{"int16", types.Typ[types.Int16], parameters.ScalarInt16, false},
		{"int32", types.Typ[types.Int32], parameters.ScalarInt32, false},
		{"int64", types.Typ[types.Int64], parameters.ScalarInt64, false},
		{"uint", types.Typ[types.Uint], parameters.ScalarUint, false},
		{"uint8", types.Typ[types.Uint8], parameters.ScalarUint8, false},
		{"uint16", types.Typ[types.Uint16], parameters.ScalarUint16, false},
		{"uint32", types.Typ[types.Uint32], parameters.ScalarUint32, false},
		{"uint64", types.Typ[types.Uint64], parameters.ScalarUint64, false},
		{"float32", types.Typ[types.Float32], parameters.ScalarFloat32, false},
		{"float64", types.Typ[types.Float64], parameters.ScalarFloat64, false},
		{"time.Duration", duration, parameters.ScalarDuration, false},
		{"time.Time", timeTime, parameters.ScalarTime, false},
		{"named int", namedPort, parameters.ScalarInt, true},
		{"named int64 underlying", namedTimeout, parameters.ScalarInt64, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kind, needConv, err := ParamScalarKind(tt.target)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if kind != tt.kind || needConv != tt.needConv {
				t.Fatalf("got (%v, %v), want (%v, %v)", kind, needConv, tt.kind, tt.needConv)
			}
		})
	}
}

func TestParamScalarKindUnsupported(t *testing.T) {
	targets := []struct {
		name   string
		target types.Type
	}{
		{"uintptr", types.Typ[types.Uintptr]},
		{"struct", types.NewStruct(nil, nil)},
		{"pointer", types.NewPointer(types.Typ[types.String])},
		{"named struct", newNamedType("example.com/app", "app", "Config", types.NewStruct(nil, nil))},
	}
	for _, tt := range targets {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ParamScalarKind(tt.target)
			if err == nil || !strings.Contains(err.Error(), "unsupported target type") {
				t.Fatalf("expected unsupported target type error, got %v", err)
			}
		})
	}
}
