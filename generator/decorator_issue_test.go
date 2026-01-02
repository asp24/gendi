package generator

import (
	"strings"
	"testing"

	"github.com/asp24/gendi"
)

func ptr[T any](v T) *T {
	return &v
}

func TestDecoratorOnAlias(t *testing.T) {
	cfg := &di.Config{
		Services: map[string]di.Service{
			"svc": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewServiceBase",
				},
				Public: true,
			},
			"svc.alias": {
				Alias:  "svc",
				Public: true,
			},
			"svc.decorator": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewServiceDecoratorA",
					Args: []di.Argument{
						{Kind: di.ArgInner},
					},
				},
				Decorates:          "svc.alias",
				DecorationPriority: 10,
			},
		},
	}

	gen := New(cfg, testOptions(t))
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	out := string(code)

	// Verify that both base and decorator constructors are used in generated code
	if !strings.Contains(out, "NewServiceBase(") {
		t.Fatalf("expected generated code to build underlying service for decorated alias")
	}
	if !strings.Contains(out, "NewServiceDecoratorA(") {
		t.Fatalf("expected generated code to build decorator")
	}
}

func TestDecoratorWithPublicTagHasPrivateGetter(t *testing.T) {
	cfg := &di.Config{
		Tags: map[string]di.Tag{
			"public.tag": {
				ElementType: "github.com/asp24/gendi/generator/testdata/app.Service",
				Public:      true,
			},
		},
		Services: map[string]di.Service{
			"svc": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewServiceBase",
				},
			},
			"svc.decorator": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewServiceDecoratorA",
					Args: []di.Argument{
						{Kind: di.ArgInner},
					},
				},
				Decorates:          "svc",
				DecorationPriority: 10,
				Tags: []di.ServiceTag{
					{Name: "public.tag"},
				},
			},
		},
	}

	gen := New(cfg, testOptions(t))
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	out := string(code)

	// Verify that the private getter for the decorator exists (required by tag getter)
	if !strings.Contains(out, "func (c *Container) getSvcDecorator()") {
		t.Fatalf("expected private getter for tagged decorator")
	}
	// Verify that the tag getter calls the private getter of the decorator
	if !strings.Contains(out, "getSvcDecorator()") {
		t.Fatalf("expected tag getter to call decorator private getter")
	}
}

func TestDecoratorSharesStorageWithBase(t *testing.T) {
	cfg := &di.Config{
		Services: map[string]di.Service{
			"svc": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewServiceBase",
				},
				Public: true,
				Shared: ptr(true),
			},
			"svc.decorator": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewServiceDecoratorA",
					Args: []di.Argument{
						{Kind: di.ArgInner},
					},
				},
				Decorates:          "svc",
				DecorationPriority: 10,
				Public:             true,
				Shared:             ptr(true),
			},
		},
	}

	gen := New(cfg, testOptions(t))
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	out := string(code)

	// Verify that storage is attached to the decorator (base is an alias)
	if !strings.Contains(out, "svc_svc_decorator ") {
		t.Fatalf("expected storage field for decorator, got:\n%s", out)
	}
	if strings.Contains(out, "svc_svc ") {
		t.Fatalf("unexpected storage field for base alias")
	}

	// Verify that BOTH getters share the same storage
	// getSvcDecorator should use svc_svc_decoratorInit
	if !strings.Contains(out, "if c.svc_svc_decoratorInit") {
		t.Fatalf("expected decorator getter to use shared field init flag")
	}
	// getSvc should delegate to getSvcDecorator
	if !strings.Contains(out, "return c.getSvcDecorator()") {
		t.Fatalf("expected getSvc to delegate to getSvcDecorator")
	}
	// Only getSvcDecorator should check the flag directly (deduplication)
	count := strings.Count(out, "if c.svc_svc_decoratorInit")
	if count != 1 {
		t.Fatalf("expected init flag check to appear exactly once (in decorator getter), found %d", count)
	}
}
