package ir

import (
	"go/types"
	"testing"
)

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
