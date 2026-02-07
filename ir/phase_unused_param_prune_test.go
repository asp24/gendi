package ir

import (
	"testing"

	di "github.com/asp24/gendi"
)

func TestPruneUnusedParamsRemovesUnreferenced(t *testing.T) {
	usedParam := &Parameter{Name: "used"}
	unusedParam := &Parameter{Name: "unused"}

	svc := &Service{
		ID: "svc",
		Constructor: &Constructor{
			Args: []*Argument{
				{Kind: ParamRefArg, Parameter: usedParam},
				{Kind: ServiceRefArg},
			},
		},
	}

	cfg := &di.Config{
		Parameters: map[string]di.Parameter{
			"used":   {},
			"unused": {},
		},
	}

	container := &Container{
		Services: map[string]*Service{svc.ID: svc},
		Parameters: map[string]*Parameter{
			usedParam.Name:   usedParam,
			unusedParam.Name: unusedParam,
		},
		tags: map[string]*Tag{},
	}

	_ = (&unusedParamPrunePhase{}).Apply(cfg, container)

	if _, ok := container.Parameters["used"]; !ok {
		t.Fatal("expected used parameter to remain in container")
	}
	if _, ok := container.Parameters["unused"]; ok {
		t.Fatal("expected unused parameter to be removed from container")
	}
	if _, ok := cfg.Parameters["used"]; !ok {
		t.Fatal("expected used parameter to remain in cfg")
	}
	if _, ok := cfg.Parameters["unused"]; ok {
		t.Fatal("expected unused parameter to be removed from cfg")
	}
}

func TestPruneUnusedParamsKeepsAllWhenAllUsed(t *testing.T) {
	p1 := &Parameter{Name: "p1"}
	p2 := &Parameter{Name: "p2"}

	svc := &Service{
		ID: "svc",
		Constructor: &Constructor{
			Args: []*Argument{
				{Kind: ParamRefArg, Parameter: p1},
				{Kind: ParamRefArg, Parameter: p2},
			},
		},
	}

	cfg := &di.Config{
		Parameters: map[string]di.Parameter{"p1": {}, "p2": {}},
	}

	container := &Container{
		Services:   map[string]*Service{svc.ID: svc},
		Parameters: map[string]*Parameter{p1.Name: p1, p2.Name: p2},
		tags:       map[string]*Tag{},
	}

	_ = (&unusedParamPrunePhase{}).Apply(cfg, container)

	if len(container.Parameters) != 2 {
		t.Fatalf("expected 2 container parameters, got %d", len(container.Parameters))
	}
	if len(cfg.Parameters) != 2 {
		t.Fatalf("expected 2 cfg parameters, got %d", len(cfg.Parameters))
	}
}

func TestPruneUnusedParamsRemovesAllWhenNoServices(t *testing.T) {
	cfg := &di.Config{
		Parameters: map[string]di.Parameter{"orphan": {}},
	}

	container := &Container{
		Services: map[string]*Service{
			"svc": {ID: "svc"}, // no constructor
		},
		Parameters: map[string]*Parameter{
			"orphan": {Name: "orphan"},
		},
		tags: map[string]*Tag{},
	}

	_ = (&unusedParamPrunePhase{}).Apply(cfg, container)

	if len(container.Parameters) != 0 {
		t.Fatalf("expected 0 container parameters, got %d", len(container.Parameters))
	}
	if len(cfg.Parameters) != 0 {
		t.Fatalf("expected 0 cfg parameters, got %d", len(cfg.Parameters))
	}
}
