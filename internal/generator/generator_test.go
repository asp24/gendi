package generator

import (
	"strings"
	"testing"

	"github.com/asp24/gendi"
)

func TestRequiresPublicService(t *testing.T) {
	cfg := &di.Config{
		Services: map[string]*di.Service{
			"a": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/internal/generator/testdata/app.NewA",
				},
			},
		},
	}
	gen := New(cfg, Options{Out: ".", Package: "di"}, nil)
	_, err := gen.Generate()
	if err == nil || !strings.Contains(err.Error(), "at least one public service") {
		t.Fatalf("expected public service error, got %v", err)
	}
}

func TestReachabilityAndPublicGetters(t *testing.T) {
	cfg := &di.Config{
		Services: map[string]*di.Service{
			"a": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/internal/generator/testdata/app.NewA",
				},
			},
			"b": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/internal/generator/testdata/app.NewB",
					Args: []di.Argument{
						{Kind: di.ArgServiceRef, Value: "a"},
					},
				},
				Public: true,
			},
			"unused": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/internal/generator/testdata/app.NewC",
				},
			},
		},
	}

	gen := New(cfg, Options{Out: ".", Package: "di"}, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	out := string(code)

	if !strings.Contains(out, "GetB") {
		t.Fatalf("expected public getter for b")
	}
	if strings.Contains(out, "GetA") {
		t.Fatalf("unexpected public getter for private service a")
	}
	if !strings.Contains(out, "getA") {
		t.Fatalf("expected private getter for a")
	}
	if strings.Contains(out, "buildUnused") || strings.Contains(out, "getUnused") || strings.Contains(out, "GetUnused") {
		t.Fatalf("unexpected generation for unreachable service")
	}
	if strings.Contains(out, "svc_unused") {
		t.Fatalf("unexpected field for unreachable service")
	}
}
