package yaml

import (
	"testing"

	di "github.com/asp24/gendi"
	"gopkg.in/yaml.v3"
)

func TestParseServiceAlias(t *testing.T) {
	raw := &RawService{
		Alias: "@foo",
	}
	p := NewParser()
	svc, err := p.convertService(raw)
	if err != nil {
		t.Fatalf("convertService failed: %v", err)
	}
	if svc.Alias != "foo" {
		t.Errorf("expected alias 'foo', got '%s'", svc.Alias)
	}
}

func TestParseServiceAliasDirect(t *testing.T) {
	raw := &RawService{
		Alias: "foo", // direct ID, no @
	}
	p := NewParser()
	svc, err := p.convertService(raw)
	if err != nil {
		t.Fatalf("convertService failed: %v", err)
	}
	if svc.Alias != "foo" {
		t.Errorf("expected alias 'foo', got '%s'", svc.Alias)
	}
}

func TestParseArgumentReference(t *testing.T) {
	val := "@myService"
	raw := &RawArgument{
		Value: &val,
	}
	p := NewParser()
	arg, err := p.convertArgument(raw)
	if err != nil {
		t.Fatalf("convertArgument failed: %v", err)
	}
	if arg.Kind != di.ArgServiceRef {
		t.Errorf("expected kind ArgServiceRef, got %v", arg.Kind)
	}
	if arg.Value != "myService" {
		t.Errorf("expected value 'myService', got '%s'", arg.Value)
	}
}

func TestParseArgumentLiteralString(t *testing.T) {
	val := "just a string"
	raw := &RawArgument{
		Value: &val,
	}
	p := NewParser()
	arg, err := p.convertArgument(raw)
	if err != nil {
		t.Fatalf("convertArgument failed: %v", err)
	}
	if arg.Kind != di.ArgLiteral {
		t.Errorf("expected kind ArgLiteral, got %v", arg.Kind)
	}
	if arg.Literal.String() != val {
		t.Errorf("expected literal value '%s', got '%s'", val, arg.Literal.String())
	}
}

func TestParseArgumentLiteralNode(t *testing.T) {
	node := yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!int",
		Value: "42",
	}
	raw := &RawArgument{
		Node: &node,
	}
	p := NewParser()
	arg, err := p.convertArgument(raw)
	if err != nil {
		t.Fatalf("convertArgument failed: %v", err)
	}
	if arg.Kind != di.ArgLiteral {
		t.Errorf("expected kind ArgLiteral, got %v", arg.Kind)
	}
	// Verify integer parsing logic if possible
}
