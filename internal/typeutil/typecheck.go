package typeutil

import "go/types"

// IsDuration returns true if t is time.Duration (optionally behind pointers).
func IsDuration(t types.Type) bool {
	// Unwrap any pointer layers
	for {
		ptr, ok := t.(*types.Pointer)
		if !ok {
			break
		}
		t = ptr.Elem()
	}
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	if obj == nil || obj.Name() != "Duration" {
		return false
	}
	pkg := obj.Pkg()
	return pkg != nil && pkg.Path() == "time"
}

// IsString returns true if t's underlying type is the built-in string type.
func IsString(t types.Type) bool {
	return types.Identical(t.Underlying(), types.Typ[types.String])
}

// IsInt returns true if t's underlying type is the built-in int type.
func IsInt(t types.Type) bool {
	return types.Identical(t.Underlying(), types.Typ[types.Int])
}

// IsBool returns true if t's underlying type is the built-in bool type.
func IsBool(t types.Type) bool {
	return types.Identical(t.Underlying(), types.Typ[types.Bool])
}

// IsFloat64 returns true if t's underlying type is the built-in float64 type.
func IsFloat64(t types.Type) bool {
	return types.Identical(t.Underlying(), types.Typ[types.Float64])
}
