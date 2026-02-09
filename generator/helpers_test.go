package generator

import (
	"go/types"
	"testing"

	di "github.com/asp24/gendi"
)

func TestCollectPackagePathsWithTypeArgs(t *testing.T) {
	cfg := &di.Config{
		Services: map[string]di.Service{
			"events": {
				Constructor: di.Constructor{
					Func: "github.com/utils.NewChan[github.com/events.Event]",
				},
			},
			"map_svc": {
				Constructor: di.Constructor{
					Func: "github.com/utils.NewMap[string, github.com/models.User]",
				},
			},
			"nested": {
				Constructor: di.Constructor{
					Func: "github.com/utils.NewSlice[chan github.com/msgs.Message]",
				},
			},
		},
	}

	refreshPackages(cfg)
	paths := collectPackagePaths(cfg)

	expected := map[string]bool{
		"github.com/utils":  true,
		"github.com/events": true,
		"github.com/models": true,
		"github.com/msgs":   true,
	}

	for _, p := range paths {
		if !expected[p] {
			t.Errorf("unexpected package: %s", p)
		}
		delete(expected, p)
	}

	for p := range expected {
		t.Errorf("missing expected package: %s", p)
	}
}

func TestCollectPackagePathsWithGenericTypes(t *testing.T) {
	cfg := &di.Config{
		Services: map[string]di.Service{
			"pool": {
				// Generic type in Type field with type argument from different package
				Type: "*github.com/containers.Pool[github.com/models.User]",
				Constructor: di.Constructor{
					Func: "github.com/containers.NewPool[github.com/models.User]",
				},
			},
			"nested": {
				// Nested generic type
				Type: "github.com/outer.Box[github.com/inner.Item[github.com/deep.Value]]",
				Constructor: di.Constructor{
					Func: "github.com/outer.NewBox[github.com/inner.Item[github.com/deep.Value]]",
				},
			},
		},
	}

	refreshPackages(cfg)
	paths := collectPackagePaths(cfg)

	expected := map[string]bool{
		"github.com/containers": true,
		"github.com/models":     true,
		"github.com/outer":      true,
		"github.com/inner":      true,
		"github.com/deep":       true,
	}

	for _, p := range paths {
		if !expected[p] {
			t.Errorf("unexpected package: %s", p)
		}
		delete(expected, p)
	}

	for p := range expected {
		t.Errorf("missing expected package: %s", p)
	}
}

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
