package typeres

import (
	"reflect"
	"sort"
	"testing"
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
			got := CollectTypePackages(tt.typeStr)
			sort.Strings(got)
			sort.Strings(tt.want)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CollectTypePackages(%q) = %v, want %v", tt.typeStr, got, tt.want)
			}
		})
	}
}
