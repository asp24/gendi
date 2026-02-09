package generator

import (
	"go/types"
	"testing"
)

func TestIsNilable(t *testing.T) {
	tint := types.Typ[types.Int]
	tptr := types.NewPointer(tint)

	// type MyInt int
	namedInt := types.NewNamed(
		types.NewTypeName(0, nil, "MyInt", nil),
		tint,
		nil,
	)

	// type MyPtr *int
	namedPtr := types.NewNamed(
		types.NewTypeName(0, nil, "MyPtr", nil),
		tptr,
		nil,
	)

	// type AliasPtr = *int
	aliasPtr := types.NewAlias(
		types.NewTypeName(0, nil, "AliasPtr", nil),
		tptr,
	)

	tests := []struct {
		name string
		typ  types.Type
		want bool
	}{
		{"int", tint, false},
		{"*int", tptr, true},
		{"named int", namedInt, false},
		{"named ptr", namedPtr, true},
		{"alias ptr", aliasPtr, true},
		{"*interface", types.NewPointer(types.NewInterfaceType(nil, nil).Complete()), true},
		{"nil type", nil, false},
		{"interface", types.NewInterfaceType(nil, nil).Complete(), true},
		{"slice", types.NewSlice(tint), true},
		{"map", types.NewMap(tint, tint), true},
		{"chan", types.NewChan(types.SendRecv, tint), true},
		{"func", types.NewSignatureType(nil, nil, nil, nil, nil, false), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNilable(tt.typ); got != tt.want {
				t.Errorf("isNilable(%s) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}
