package ir

import (
	"go/token"
	"go/types"
	"strings"
	"testing"
)

func newNamedType(pkgPath, pkgName, typeName string, underlying types.Type) types.Type {
	pkg := types.NewPackage(pkgPath, pkgName)
	obj := types.NewTypeName(token.NoPos, pkg, typeName, nil)
	return types.NewNamed(obj, underlying, nil)
}

func TestCasterMethod(t *testing.T) {
	duration := newNamedType("time", "time", "Duration", types.Typ[types.Int64])
	timeTime := newNamedType("time", "time", "Time", types.NewStruct(nil, nil))
	namedPort := newNamedType("example.com/app", "app", "Port", types.Typ[types.Int])
	// A named type defined as `type Timeout time.Duration` has underlying
	// int64, so it routes through ToInt64 plus a static conversion.
	namedTimeout := newNamedType("example.com/app", "app", "Timeout", types.Typ[types.Int64])

	tests := []struct {
		name     string
		target   types.Type
		method   string
		needConv bool
	}{
		{"string", types.Typ[types.String], "ToString", false},
		{"bool", types.Typ[types.Bool], "ToBool", false},
		{"int", types.Typ[types.Int], "ToInt", false},
		{"int8", types.Typ[types.Int8], "ToInt8", false},
		{"int16", types.Typ[types.Int16], "ToInt16", false},
		{"int32", types.Typ[types.Int32], "ToInt32", false},
		{"int64", types.Typ[types.Int64], "ToInt64", false},
		{"uint", types.Typ[types.Uint], "ToUint", false},
		{"uint8", types.Typ[types.Uint8], "ToUint8", false},
		{"uint16", types.Typ[types.Uint16], "ToUint16", false},
		{"uint32", types.Typ[types.Uint32], "ToUint32", false},
		{"uint64", types.Typ[types.Uint64], "ToUint64", false},
		{"float32", types.Typ[types.Float32], "ToFloat32", false},
		{"float64", types.Typ[types.Float64], "ToFloat64", false},
		{"time.Duration", duration, "ToDuration", false},
		{"time.Time", timeTime, "ToTime", false},
		{"named int", namedPort, "ToInt", true},
		{"named int64 underlying", namedTimeout, "ToInt64", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method, needConv, err := CasterMethod(tt.target)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if method != tt.method || needConv != tt.needConv {
				t.Fatalf("got (%s, %v), want (%s, %v)", method, needConv, tt.method, tt.needConv)
			}
		})
	}
}

func TestCasterMethodUnsupported(t *testing.T) {
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
			_, _, err := CasterMethod(tt.target)
			if err == nil || !strings.Contains(err.Error(), "unsupported target type") {
				t.Fatalf("expected unsupported target type error, got %v", err)
			}
		})
	}
}
