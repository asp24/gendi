package ir

import (
	"go/types"
	"strings"
	"testing"

	di "github.com/asp24/gendi"
)

func TestTaggedElementTypeAssignable(t *testing.T) {
	container := NewContainer()
	r := &argResolver{}
	arg := di.Argument{Kind: di.ArgTagged, Value: "tag.test"}

	if _, err := r.resolve(container, "svc.one", 0, arg, types.NewSlice(types.Typ[types.Int])); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	emptyIface := types.NewInterfaceType(nil, nil)
	emptyIface.Complete()
	if _, err := r.resolve(container, "svc.two", 0, arg, types.NewSlice(emptyIface)); err != nil {
		t.Fatalf("expected assignable element type, got %v", err)
	}
}

func TestTaggedElementTypeNotAssignable(t *testing.T) {
	container := NewContainer()
	r := &argResolver{}
	arg := di.Argument{Kind: di.ArgTagged, Value: "tag.test"}

	emptyIface := types.NewInterfaceType(nil, nil)
	emptyIface.Complete()
	if _, err := r.resolve(container, "svc.one", 0, arg, types.NewSlice(emptyIface)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err := r.resolve(container, "svc.two", 0, arg, types.NewSlice(types.Typ[types.Int]))
	if err == nil || !strings.Contains(err.Error(), "not assignable") {
		t.Fatalf("expected element type mismatch error, got %v", err)
	}
}
