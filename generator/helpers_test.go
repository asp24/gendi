package generator

import (
	"reflect"
	"sort"
	"testing"

	di "github.com/asp24/gendi"
)

func TestCollectTypePackages(t *testing.T) {
	tests := []struct {
		typeStr string
		want    []string
	}{
		// Basic types
		{"int", nil},
		{"string", nil},
		{"bool", nil},

		// Named types
		{"github.com/pkg.Type", []string{"github.com/pkg"}},

		// Pointer types
		{"*github.com/pkg.Type", []string{"github.com/pkg"}},
		{"**github.com/pkg.Type", []string{"github.com/pkg"}},

		// Slice types
		{"[]github.com/pkg.Type", []string{"github.com/pkg"}},
		{"[][]github.com/pkg.Type", []string{"github.com/pkg"}},

		// Array types
		{"[10]github.com/pkg.Type", []string{"github.com/pkg"}},

		// Channel types
		{"chan github.com/pkg.Type", []string{"github.com/pkg"}},
		{"<-chan github.com/pkg.Type", []string{"github.com/pkg"}},
		{"chan<- github.com/pkg.Type", []string{"github.com/pkg"}},

		// Map types
		{"map[string]github.com/pkg.Type", []string{"github.com/pkg"}},
		{"map[github.com/key.K]github.com/val.V", []string{"github.com/key", "github.com/val"}},

		// Nested composite types
		{"chan []github.com/pkg.Type", []string{"github.com/pkg"}},
		{"*[]chan github.com/pkg.Type", []string{"github.com/pkg"}},

		// Generic named types with type arguments
		{"github.com/pkg.Box[github.com/other.T]", []string{"github.com/pkg", "github.com/other"}},
		{"github.com/pkg.Map[string, github.com/val.V]", []string{"github.com/pkg", "github.com/val"}},
		{"github.com/pkg.Map[github.com/key.K, github.com/val.V]", []string{"github.com/key", "github.com/pkg", "github.com/val"}},

		// Pointer to generic type
		{"*github.com/pkg.Box[github.com/other.T]", []string{"github.com/pkg", "github.com/other"}},

		// Generic type with composite type argument
		{"github.com/pkg.Chan[chan github.com/events.Event]", []string{"github.com/events", "github.com/pkg"}},
		{"github.com/pkg.Slice[[]github.com/items.Item]", []string{"github.com/items", "github.com/pkg"}},

		// Nested generic types
		{"github.com/outer.Box[github.com/inner.Box[github.com/deep.T]]", []string{"github.com/deep", "github.com/inner", "github.com/outer"}},
	}

	for _, tt := range tests {
		t.Run(tt.typeStr, func(t *testing.T) {
			got := collectTypePackages(tt.typeStr)
			sort.Strings(got)
			sort.Strings(tt.want)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("collectTypePackages(%q) = %v, want %v", tt.typeStr, got, tt.want)
			}
		})
	}
}

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

	paths, err := collectPackagePaths(cfg)
	if err != nil {
		t.Fatalf("collectPackagePaths failed: %v", err)
	}

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

	paths, err := collectPackagePaths(cfg)
	if err != nil {
		t.Fatalf("collectPackagePaths failed: %v", err)
	}

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
