package pipeline

import (
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
