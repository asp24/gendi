package ir

import (
	"go/types"
	"testing"

	di "github.com/gendi-org/gendi"
)

func TestDependencyBuilderCollectsUniqueDependencies(t *testing.T) {
	stringType := types.Typ[types.String]

	dep := &Service{ID: "dep", Type: stringType}
	other := &Service{ID: "other", Type: stringType}
	app := &Service{
		ID: "app",
		Constructor: &Constructor{
			Kind: FuncConstructor,
			Args: []*Argument{
				{Kind: ServiceRefArg, Service: dep},
				{Kind: ServiceRefArg, Service: dep},
				{Kind: FieldAccessArg, FieldAccess: &FieldAccess{Service: dep, FieldNames: []string{"X"}}},
				{Kind: ServiceRefArg, Service: other},
			},
		},
	}
	alias := &Service{ID: "alias", Alias: dep}

	container := NewContainer()
	container.Services["dep"] = dep
	container.Services["other"] = other
	container.Services["app"] = app
	container.Services["alias"] = alias

	if err := (&dependencyBuilderPhase{}).Apply(di.NewConfig(), container); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := len(app.Dependencies); got != 2 {
		t.Fatalf("expected 2 unique dependencies, got %d", got)
	}
	if got := app.dependencyRefCount("dep"); got != 3 {
		t.Errorf("expected 3 references to dep, got %d", got)
	}
	if got := app.dependencyRefCount("other"); got != 1 {
		t.Errorf("expected 1 reference to other, got %d", got)
	}
	if got := alias.dependencyRefCount("dep"); got != 1 {
		t.Errorf("expected alias to count 1 reference to dep, got %d", got)
	}
}
