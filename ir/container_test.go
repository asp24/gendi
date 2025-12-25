package ir

import (
	"go/types"
	"testing"
)

func TestContainerDecoratorsByBase(t *testing.T) {
	base := &Service{ID: "base"}
	decA := &Service{ID: "decA"}
	decB := &Service{ID: "decB"}
	base.Decorators = []*Service{decA, decB}

	container := &Container{
		Services: map[string]*Service{
			base.ID: base,
			decA.ID: decA,
			decB.ID: decB,
		},
	}

	got := container.DecoratorsByBase()
	decs := got["base"]
	if len(decs) != 2 || decs[0].ID != "decA" || decs[1].ID != "decB" {
		t.Fatalf("unexpected decorators: %#v", decs)
	}

	// Ensure the returned slice doesn't alias the original.
	decC := &Service{ID: "decC"}
	decs[0] = decC
	if base.Decorators[0].ID != "decA" {
		t.Fatalf("unexpected aliasing in decorators slice")
	}
}

func TestContainerBaseByDecorator(t *testing.T) {
	base := &Service{ID: "base"}
	dec := &Service{ID: "dec", Decorates: base}
	container := &Container{
		Services: map[string]*Service{
			base.ID: base,
			dec.ID:  dec,
		},
	}

	got := container.BaseByDecorator()
	if got["dec"] != "base" {
		t.Fatalf("expected base for decorator, got: %#v", got)
	}
	if _, ok := got["base"]; ok {
		t.Fatalf("did not expect base to be a decorator key")
	}
}

func TestContainerParamGetters(t *testing.T) {
	customPkg := types.NewPackage("example.com/custom", "custom")
	customType := types.NewNamed(types.NewTypeName(0, customPkg, "Thing", nil), nil, nil)

	params := map[string]*Parameter{
		"str":   {Name: "str", Type: types.Typ[types.String]},
		"int":   {Name: "int", Type: types.Typ[types.Int]},
		"extra": {Name: "extra", Type: customType},
	}

	svc := &Service{
		ID: "svc",
		Constructor: &Constructor{
			Args: []*Argument{
				{Kind: ParamRefArg, Parameter: params["int"]},
				{Kind: ParamRefArg, Parameter: params["extra"]},
			},
		},
	}

	container := &Container{
		Parameters: params,
		Services:   map[string]*Service{"svc": svc},
	}

	got := container.ParamGetters()
	if got["str"] != "GetString" {
		t.Fatalf("expected string getter, got: %#v", got["str"])
	}
	if got["int"] != "GetInt" {
		t.Fatalf("expected int getter, got: %#v", got["int"])
	}
	if _, ok := got["extra"]; ok {
		t.Fatalf("did not expect getter for custom type")
	}
}
