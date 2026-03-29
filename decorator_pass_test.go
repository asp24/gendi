package di

import (
	"strings"
	"testing"
)

func TestDecoratorPassSingleDecorator(t *testing.T) {
	cfg := &Config{
		Services: map[string]Service{
			"base": {
				Autoconfigure: true,
				Constructor: Constructor{
					Func: "app.NewBase",
				},
			},
			"decorator": {
				Decorates:          "base",
				DecorationPriority: 10,
				Constructor: Constructor{
					Func: "app.NewDecorator",
					Args: []Argument{
						{Kind: ArgInner},
					},
				},
			},
		},
	}

	pass := &DecoratorPass{}
	result, err := pass.Process(cfg)
	if err != nil {
		t.Fatalf("DecoratorPass failed: %v", err)
	}

	// Should have 3 services: decorator.inner, decorator, base (alias)
	if len(result.Services) != 3 {
		t.Fatalf("expected 3 services, got %d", len(result.Services))
	}

	// Check inner service
	innerSvc, ok := result.Services["decorator.inner"]
	if !ok {
		t.Fatalf("expected decorator.inner service to be created")
	}
	if innerSvc.Constructor.Func != "app.NewBase" {
		t.Fatalf("expected inner service to have original constructor")
	}
	if innerSvc.Public {
		t.Fatalf("expected inner service to not be public")
	}
	if innerSvc.Autoconfigure {
		t.Fatalf("expected inner service to have autoconfigure disabled")
	}

	// Check base becomes alias
	baseSvc := result.Services["base"]
	if baseSvc.Alias != "decorator" {
		t.Fatalf("expected base to become alias to decorator, got: %q", baseSvc.Alias)
	}
	if baseSvc.Constructor.Func != "" {
		t.Fatalf("expected base constructor to be cleared")
	}

	// Check decorator args are rewritten
	decSvc := result.Services["decorator"]
	if len(decSvc.Constructor.Args) != 1 {
		t.Fatalf("expected decorator to have 1 arg")
	}
	arg := decSvc.Constructor.Args[0]
	if arg.Kind != ArgServiceRef {
		t.Fatalf("expected @.inner to be rewritten to service ref")
	}
	if arg.Value != "decorator.inner" {
		t.Fatalf("expected arg to reference decorator.inner, got: %q", arg.Value)
	}
	if decSvc.Decorates != "" {
		t.Fatalf("expected Decorates field to be cleared")
	}
}

func TestDecoratorPassChain(t *testing.T) {
	cfg := &Config{
		Services: map[string]Service{
			"base": {
				Constructor: Constructor{
					Func: "app.NewBase",
				},
			},
			"decA": {
				Decorates:          "base",
				DecorationPriority: 10,
				Constructor: Constructor{
					Func: "app.NewDecoratorA",
					Args: []Argument{{Kind: ArgInner}},
				},
			},
			"decB": {
				Decorates:          "base",
				DecorationPriority: 20,
				Constructor: Constructor{
					Func: "app.NewDecoratorB",
					Args: []Argument{{Kind: ArgInner}},
				},
			},
		},
	}

	pass := &DecoratorPass{}
	result, err := pass.Process(cfg)
	if err != nil {
		t.Fatalf("DecoratorPass failed: %v", err)
	}

	// Should have 4 services: decA.inner, decA, decB, base (alias)
	if len(result.Services) != 4 {
		t.Logf("Services found:")
		for id := range result.Services {
			t.Logf("  - %s", id)
		}
		t.Fatalf("expected 4 services, got %d", len(result.Services))
	}

	// Check inner service
	innerSvc, ok := result.Services["decA.inner"]
	if !ok {
		t.Fatalf("expected decA.inner service to be created")
	}
	if innerSvc.Constructor.Func != "app.NewBase" {
		t.Fatalf("expected inner service to have original constructor")
	}

	// Check base becomes alias to outermost decorator (decB)
	baseSvc := result.Services["base"]
	if baseSvc.Alias != "decB" {
		t.Fatalf("expected base to be alias to decB, got: %q", baseSvc.Alias)
	}

	// Check decA references inner
	decA := result.Services["decA"]
	if len(decA.Constructor.Args) != 1 {
		t.Fatalf("expected decA to have 1 arg")
	}
	if decA.Constructor.Args[0].Value != "decA.inner" {
		t.Fatalf("expected decA to reference decA.inner, got: %q", decA.Constructor.Args[0].Value)
	}

	// Check decB references decA
	decB := result.Services["decB"]
	if len(decB.Constructor.Args) != 1 {
		t.Fatalf("expected decB to have 1 arg")
	}
	if decB.Constructor.Args[0].Value != "decA" {
		t.Fatalf("expected decB to reference decA, got: %q", decB.Constructor.Args[0].Value)
	}
}

func TestDecoratorPassEqualPriorityUsesIDTieBreak(t *testing.T) {
	cfg := &Config{
		Services: map[string]Service{
			"base": {
				Constructor: Constructor{
					Func: "app.NewBase",
				},
			},
			"decorator.a": {
				Decorates:          "base",
				DecorationPriority: 10,
				Constructor: Constructor{
					Func: "app.NewDecoratorA",
					Args: []Argument{{Kind: ArgInner}},
				},
			},
			"decorator.b": {
				Decorates:          "base",
				DecorationPriority: 10,
				Constructor: Constructor{
					Func: "app.NewDecoratorB",
					Args: []Argument{{Kind: ArgInner}},
				},
			},
		},
	}

	pass := &DecoratorPass{}
	result, err := pass.Process(cfg)
	if err != nil {
		t.Fatalf("DecoratorPass failed: %v", err)
	}

	baseSvc := result.Services["base"]
	if baseSvc.Alias != "decorator.b" {
		t.Fatalf("expected base to be alias to lexicographically later outer decorator, got: %q", baseSvc.Alias)
	}

	decA := result.Services["decorator.a"]
	if len(decA.Constructor.Args) != 1 {
		t.Fatalf("expected decorator.a to have 1 arg")
	}
	if decA.Constructor.Args[0].Value != "decorator.a.inner" {
		t.Fatalf("expected decorator.a to reference its inner service first, got: %q", decA.Constructor.Args[0].Value)
	}

	decB := result.Services["decorator.b"]
	if len(decB.Constructor.Args) != 1 {
		t.Fatalf("expected decorator.b to have 1 arg")
	}
	if decB.Constructor.Args[0].Value != "decorator.a" {
		t.Fatalf("expected decorator.b to wrap decorator.a on equal priority tie, got: %q", decB.Constructor.Args[0].Value)
	}
}

func TestDecoratorPassSharedPropagation(t *testing.T) {
	cfg := &Config{
		Services: map[string]Service{
			"base": {
				Shared: false,
				Constructor: Constructor{
					Func: "app.NewBase",
				},
			},
			"decorator": {
				Shared:             true,
				Decorates:          "base",
				DecorationPriority: 10,
				Constructor: Constructor{
					Func: "app.NewDecorator",
					Args: []Argument{{Kind: ArgInner}},
				},
			},
		},
	}

	pass := &DecoratorPass{}
	result, err := pass.Process(cfg)
	if err != nil {
		t.Fatalf("DecoratorPass failed: %v", err)
	}

	// Decorator and alias should become shared, but not inner
	for id, svc := range result.Services {
		if id == "decorator.inner" {
			// Inner should keep its original non-shared value
			if svc.Shared {
				t.Fatalf("expected inner service to remain non-shared")
			}
		} else {
			// Decorator and base (alias) should be shared
			if !svc.Shared {
				t.Fatalf("expected service %q to be shared", id)
			}
		}
	}
}

func TestDecoratorPassCycleDetection(t *testing.T) {
	// Cycle A -> B -> C -> A will be caught by "cannot be decorated" check
	// When processing C (decorates A), it will find that A.Decorates="b" (non-empty)
	cfg := &Config{
		Services: map[string]Service{
			"a": {
				Decorates: "b",
				Constructor: Constructor{
					Func: "app.NewA",
				},
			},
			"b": {
				Decorates: "c",
				Constructor: Constructor{
					Func: "app.NewB",
				},
			},
			"c": {
				Decorates: "a",
				Constructor: Constructor{
					Func: "app.NewC",
				},
			},
		},
	}

	pass := &DecoratorPass{}
	_, err := pass.Process(cfg)
	if err == nil {
		t.Fatalf("expected cycle detection error")
	}
	// Cycles are caught by "cannot be decorated" check
	if !strings.Contains(err.Error(), "cannot be decorated") {
		t.Fatalf("expected cannot-be-decorated error for cycle, got: %v", err)
	}
}

func TestDecoratorPassDecoratorCannotBeDecorated(t *testing.T) {
	cfg := &Config{
		Services: map[string]Service{
			"base": {
				Constructor: Constructor{
					Func: "app.NewBase",
				},
			},
			"decA": {
				Decorates: "base",
				Constructor: Constructor{
					Func: "app.NewDecoratorA",
				},
			},
			"decB": {
				Decorates: "decA",
				Constructor: Constructor{
					Func: "app.NewDecoratorB",
				},
			},
		},
	}

	pass := &DecoratorPass{}
	_, err := pass.Process(cfg)
	if err == nil {
		t.Fatalf("expected error when decorating decorator")
	}
	if !strings.Contains(err.Error(), "cannot be decorated") {
		t.Fatalf("expected 'cannot be decorated' error, got: %v", err)
	}
}

func TestDecoratorPassUnknownBase(t *testing.T) {
	cfg := &Config{
		Services: map[string]Service{
			"decorator": {
				Decorates: "unknown",
				Constructor: Constructor{
					Func: "app.NewDecorator",
				},
			},
		},
	}

	pass := &DecoratorPass{}
	_, err := pass.Process(cfg)
	if err == nil {
		t.Fatalf("expected error when decorating unknown service")
	}
	if !strings.Contains(err.Error(), "unknown service") {
		t.Fatalf("expected 'unknown service' error, got: %v", err)
	}
}

func TestDecoratorPassNoDecorators(t *testing.T) {
	cfg := &Config{
		Services: map[string]Service{
			"service1": {
				Constructor: Constructor{
					Func: "app.NewService1",
				},
			},
			"service2": {
				Constructor: Constructor{
					Func: "app.NewService2",
				},
			},
		},
	}

	pass := &DecoratorPass{}
	result, err := pass.Process(cfg)
	if err != nil {
		t.Fatalf("DecoratorPass failed: %v", err)
	}

	// Should remain unchanged
	if len(result.Services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(result.Services))
	}
}

func TestDecoratorPassMultipleArgsWithInner(t *testing.T) {
	cfg := &Config{
		Services: map[string]Service{
			"base": {
				Constructor: Constructor{
					Func: "app.NewBase",
				},
			},
			"decorator": {
				Decorates:          "base",
				DecorationPriority: 10,
				Constructor: Constructor{
					Func: "app.NewDecorator",
					Args: []Argument{
						{Kind: ArgServiceRef, Value: "logger"},
						{Kind: ArgInner},
						{Kind: ArgParam, Value: "timeout"},
					},
				},
			},
		},
	}

	pass := &DecoratorPass{}
	result, err := pass.Process(cfg)
	if err != nil {
		t.Fatalf("DecoratorPass failed: %v", err)
	}

	decSvc := result.Services["decorator"]
	if len(decSvc.Constructor.Args) != 3 {
		t.Fatalf("expected 3 args, got %d", len(decSvc.Constructor.Args))
	}

	// First arg unchanged
	if decSvc.Constructor.Args[0].Kind != ArgServiceRef || decSvc.Constructor.Args[0].Value != "logger" {
		t.Fatalf("expected first arg to be @logger")
	}

	// Second arg rewritten
	if decSvc.Constructor.Args[1].Kind != ArgServiceRef || decSvc.Constructor.Args[1].Value != "decorator.inner" {
		t.Fatalf("expected second arg to be @decorator.inner")
	}

	// Third arg unchanged
	if decSvc.Constructor.Args[2].Kind != ArgParam || decSvc.Constructor.Args[2].Value != "timeout" {
		t.Fatalf("expected third arg to be %%timeout%%")
	}
}
