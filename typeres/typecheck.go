package typeres

import "go/types"

// IsDuration returns true if t is time.Duration (or an alias of it).
// Pointers are intentionally not unwrapped: the generator emits duration
// values, which are not assignable to *time.Duration.
func IsDuration(t types.Type) bool {
	named, ok := types.Unalias(t).(*types.Named)
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

// IsTime returns true if t is time.Time (or an alias of it). Pointers and
// named types with a time.Time underlying are intentionally not accepted:
// only the exact type is a supported scalar target.
func IsTime(t types.Type) bool {
	named, ok := types.Unalias(t).(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	if obj == nil || obj.Name() != "Time" {
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
