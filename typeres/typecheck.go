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
