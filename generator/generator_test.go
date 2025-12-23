package generator

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/asp24/gendi"
)

func TestRequiresPublicService(t *testing.T) {
	cfg := &di.Config{
		Services: map[string]*di.Service{
			"a": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewA",
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
					Func: "github.com/asp24/gendi/generator/testdata/app.NewA",
				},
			},
			"b": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewB",
					Args: []di.Argument{
						{Kind: di.ArgServiceRef, Value: "a"},
					},
				},
				Public: true,
			},
			"unused": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewC",
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

func TestParameterProviderCodegen(t *testing.T) {
	cfg := &di.Config{
		Parameters: map[string]di.Parameter{
			"log_prefix": {
				Type:  "string",
				Value: mustLiteralNode("!!str", "[app] "),
			},
		},
		Services: map[string]*di.Service{
			"logger": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewLogger",
					Args: []di.Argument{
						{Kind: di.ArgParam, Value: "log_prefix"},
					},
				},
				Public: true,
			},
		},
	}

	gen := New(cfg, Options{Out: ".", Package: "di"}, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	out := string(code)
	if !strings.Contains(out, "NewContainer") {
		t.Fatalf("expected container constructor when parameters are present")
	}
	if !strings.Contains(out, "GetString(\"log_prefix\")") {
		t.Fatalf("expected parameter provider lookup in generated code")
	}
}

func TestDurationParameterCodegen(t *testing.T) {
	cfg := &di.Config{
		Parameters: map[string]di.Parameter{
			"timeout": {
				Type:  "time.Duration",
				Value: mustLiteralNode("!!str", "1s"),
			},
		},
		Services: map[string]*di.Service{
			"timer": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewTimer",
					Args: []di.Argument{
						{Kind: di.ArgParam, Value: "timeout"},
					},
				},
				Public: true,
			},
		},
	}

	gen := New(cfg, Options{Out: ".", Package: "di"}, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	out := string(code)
	if !strings.Contains(out, "GetDuration(\"timeout\")") {
		t.Fatalf("expected duration parameter lookup")
	}
}

func TestNullLiteralArgument(t *testing.T) {
	cfg := &di.Config{
		Services: map[string]*di.Service{
			"b": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewB",
					Args: []di.Argument{
						{Kind: di.ArgLiteral, Literal: mustLiteralNode("!!null", "null")},
					},
				},
				Public: true,
			},
		},
	}

	gen := New(cfg, Options{Out: ".", Package: "di"}, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if !strings.Contains(string(code), "NewB(nil)") {
		t.Fatalf("expected nil literal for null argument")
	}
}

func TestServiceAliasCodegen(t *testing.T) {
	cfg := &di.Config{
		Services: map[string]*di.Service{
			"logger": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewLogger",
					Args: []di.Argument{
						{Kind: di.ArgParam, Value: "log_prefix"},
					},
				},
				Public: true,
			},
			"logger.alias": {
				Alias:  "logger",
				Public: true,
			},
		},
		Parameters: map[string]di.Parameter{
			"log_prefix": {
				Type:  "string",
				Value: mustLiteralNode("!!str", "[app] "),
			},
		},
	}

	gen := New(cfg, Options{Out: ".", Package: "di"}, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	out := string(code)
	if !strings.Contains(out, "GetLoggerAlias") {
		t.Fatalf("expected alias public getter")
	}
	if strings.Contains(out, "buildLoggerAlias") {
		t.Fatalf("unexpected build function for alias")
	}
	if !strings.Contains(out, "return c.getLogger()") {
		t.Fatalf("expected alias getter to forward to target")
	}
}

func TestDecoratorPrivateGetterElidedWhenUnused(t *testing.T) {
	cfg := &di.Config{
		Services: map[string]*di.Service{
			"svc": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewServiceBase",
				},
				Public: true,
			},
			"svc.decoratorA": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewServiceDecoratorA",
					Args: []di.Argument{
						{Kind: di.ArgInner},
					},
				},
				Decorates:          "svc",
				DecorationPriority: 10,
			},
			"svc.decoratorB": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewServiceDecoratorB",
					Args: []di.Argument{
						{Kind: di.ArgInner},
					},
				},
				Decorates:          "svc",
				DecorationPriority: 20,
			},
		},
	}

	gen := New(cfg, Options{Out: ".", Package: "di"}, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	out := string(code)
	if strings.Contains(out, "getSvcDecoratorA") || strings.Contains(out, "getSvcDecoratorB") {
		t.Fatalf("unexpected private getters for decorators")
	}
	if strings.Contains(out, "svc_svc_decoratorA") || strings.Contains(out, "svc_svc_decoratorB") {
		t.Fatalf("unexpected fields for decorators")
	}
}

func TestDecoratorPrivateGetterGeneratedWhenReferenced(t *testing.T) {
	cfg := &di.Config{
		Services: map[string]*di.Service{
			"svc": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewServiceBase",
				},
			},
			"svc.decoratorA": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewServiceDecoratorA",
					Args: []di.Argument{
						{Kind: di.ArgInner},
					},
				},
				Decorates:          "svc",
				DecorationPriority: 10,
			},
			"consumer": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewConsumer",
					Args: []di.Argument{
						{Kind: di.ArgServiceRef, Value: "svc.decoratorA"},
					},
				},
				Public: true,
			},
		},
	}

	gen := New(cfg, Options{Out: ".", Package: "di"}, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	out := string(code)
	if !strings.Contains(out, "getSvcDecoratorA") {
		t.Fatalf("expected private getter for referenced decorator")
	}
	if !strings.Contains(out, "svc_svc_decoratorA") {
		t.Fatalf("expected field for referenced decorator")
	}
}

func TestServiceTypeAssignableOverride(t *testing.T) {
	cfg := &di.Config{
		Services: map[string]*di.Service{
			"svc": {
				Type: "github.com/asp24/gendi/generator/testdata/app.Service",
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewServiceBaseConcrete",
				},
				Public: true,
			},
		},
	}

	gen := New(cfg, Options{Out: ".", Package: "di"}, nil)
	if _, err := gen.Generate(); err != nil {
		t.Fatalf("generate failed: %v", err)
	}
}

func TestDecoratorAssignableToDeclaredBaseType(t *testing.T) {
	cfg := &di.Config{
		Services: map[string]*di.Service{
			"svc": {
				Type: "github.com/asp24/gendi/generator/testdata/app.Service",
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewServiceBaseConcrete",
				},
				Public: true,
			},
			"svc.decorator": {
				Constructor: di.Constructor{
					Func: "github.com/asp24/gendi/generator/testdata/app.NewServiceDecoratorAConcrete",
					Args: []di.Argument{
						{Kind: di.ArgInner},
					},
				},
				Decorates:          "svc",
				DecorationPriority: 10,
			},
		},
	}

	gen := New(cfg, Options{Out: ".", Package: "di"}, nil)
	if _, err := gen.Generate(); err != nil {
		t.Fatalf("generate failed: %v", err)
	}
}

func mustLiteralNode(tag, value string) yaml.Node {
	return yaml.Node{Kind: yaml.ScalarNode, Tag: tag, Value: value}
}
