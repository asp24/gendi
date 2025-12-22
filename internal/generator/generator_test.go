package generator

import (
	"strings"
	"testing"

	"github.com/asp24/gendi"
	"gopkg.in/yaml.v3"
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
					Func: "github.com/asp24/gendi/internal/generator/testdata/app.NewLogger",
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

func mustLiteralNode(tag, value string) yaml.Node {
	return yaml.Node{Kind: yaml.ScalarNode, Tag: tag, Value: value}
}
