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

	// Verify that the decorator chain for the alias correctly builds the underlying service
	if !strings.Contains(out, "buildSvc()") {
		t.Fatalf("expected generated code to build underlying service for decorated alias")
	}
	// And verify the decorator is applied
	if !strings.Contains(out, "buildSvcDecoratorDecorator") {
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

	// Verify that ONLY ONE storage field exists for both
	if !strings.Contains(out, "svc_svc ") {
		t.Fatalf("expected storage field for base, got:\n%s", out)
	}
	if strings.Contains(out, "svc_svc_decorator") {
		t.Fatalf("unexpected storage field for decorator (should share with base)")
	}

	// Verify that BOTH getters use the same field
	// getSvc should use svc_svcInit
	if !strings.Contains(out, "if c.svc_svcInit") {
		t.Fatalf("expected getSvc to use shared field init flag")
	}
	// getSvcDecorator should delegate to getSvc
	if !strings.Contains(out, "return c.getSvc()") {
		t.Fatalf("expected getSvcDecorator to delegate to getSvc")
	}
	// Only getSvc should check the flag directly (deduplication)
	count := strings.Count(out, "if c.svc_svcInit")
	if count != 1 {
		t.Fatalf("expected init flag check to appear exactly once (in root getter), found %d", count)
	}
}
