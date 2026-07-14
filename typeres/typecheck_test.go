package typeres

import (
	"go/token"
	"go/types"
	"testing"
)

func TestIsDuration(t *testing.T) {
	pkg := types.NewPackage("time", "time")
	duration := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Duration", nil),
		types.Typ[types.Int64], nil)
	alias := types.NewAlias(
		types.NewTypeName(token.NoPos, pkg, "D", nil), duration)

	if !IsDuration(duration) {
		t.Error("time.Duration must be detected")
	}
	if !IsDuration(alias) {
		t.Error("an alias of time.Duration must be detected")
	}
	// A *time.Duration parameter cannot accept the integer literal the
	// generator emits for durations.
	if IsDuration(types.NewPointer(duration)) {
		t.Error("*time.Duration must not be treated as a duration value")
	}
}
