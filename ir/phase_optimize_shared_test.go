package ir

import (
	"testing"
)

func TestOptimizeShared(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(c *Container)
		validate func(t *testing.T, c *Container)
	}{
		{
			name: "optimizes shared service used by single shared parent",
			setup: func(c *Container) {
				s := &Service{ID: "S", Shared: true, Public: false}
				p := &Service{ID: "P", Shared: true, Dependencies: []*Service{s}}
				c.Services["S"] = s
				c.Services["P"] = p
			},
			validate: func(t *testing.T, c *Container) {
				if c.Services["S"].Shared {
					t.Error("S should be non-shared")
				}
			},
		},
		{
			name: "does not optimize if parent is non-shared",
			setup: func(c *Container) {
				s := &Service{ID: "S", Shared: true, Public: false}
				p := &Service{ID: "P", Shared: false, Dependencies: []*Service{s}}
				c.Services["S"] = s
				c.Services["P"] = p
			},
			validate: func(t *testing.T, c *Container) {
				if !c.Services["S"].Shared {
					t.Error("S should remain shared")
				}
			},
		},
		{
			name: "does not optimize if used by multiple services",
			setup: func(c *Container) {
				s := &Service{ID: "S", Shared: true, Public: false}
				p1 := &Service{ID: "P1", Shared: true, Dependencies: []*Service{s}}
				p2 := &Service{ID: "P2", Shared: true, Dependencies: []*Service{s}}
				c.Services["S"] = s
				c.Services["P1"] = p1
				c.Services["P2"] = p2
			},
			validate: func(t *testing.T, c *Container) {
				if !c.Services["S"].Shared {
					t.Error("S should remain shared")
				}
			},
		},
		{
			name: "does not optimize public service",
			setup: func(c *Container) {
				s := &Service{ID: "S", Shared: true, Public: true}
				p := &Service{ID: "P", Shared: true, Dependencies: []*Service{s}}
				c.Services["S"] = s
				c.Services["P"] = p
			},
			validate: func(t *testing.T, c *Container) {
				if !c.Services["S"].Shared {
					t.Error("S should remain shared")
				}
			},
		},
		{
			name: "keeps dependency shared when direct parent is public non-shared",
			setup: func(c *Container) {
				s := &Service{
					ID:     "S",
					Shared: true,
					Public: false,
				}
				publicParent := &Service{
					ID:           "!tagged:myTag",
					Shared:       false,
					Public:       true,
					Dependencies: []*Service{s},
				}
				c.Services["S"] = s
				c.Services["!tagged:myTag"] = publicParent
			},
			validate: func(t *testing.T, c *Container) {
				if !c.Services["S"].Shared {
					t.Error("S should remain shared because its only parent is public non-shared")
				}
			},
		},
		{
			name: "does not optimize alias",
			setup: func(c *Container) {
				target := &Service{ID: "T", Shared: true}
				s := &Service{
					ID:     "S",
					Shared: true,
					Public: false,
					Alias:  target,
				}
				p := &Service{ID: "P", Shared: true, Dependencies: []*Service{s}}
				c.Services["S"] = s
				c.Services["T"] = target
				c.Services["P"] = p
			},
			validate: func(t *testing.T, c *Container) {
				if !c.Services["S"].Shared {
					t.Error("S should remain shared")
				}
			},
		},
		{
			name: "keeps parent shared when it feeds a multi-dependency service",
			setup: func(c *Container) {
				cSvc := &Service{ID: "C", Shared: true, Public: false}
				b := &Service{ID: "B", Shared: true, Public: false, Dependencies: []*Service{cSvc}}
				other := &Service{ID: "X", Shared: true, Public: false}
				p := &Service{
					ID:           "P",
					Shared:       false,
					Public:       false,
					Dependencies: []*Service{b, other},
				}

				c.Services["C"] = cSvc
				c.Services["B"] = b
				c.Services["X"] = other
				c.Services["P"] = p
			},
			validate: func(t *testing.T, c *Container) {
				if c.Services["C"].Shared {
					t.Error("C should be non-shared")
				}
				if !c.Services["B"].Shared {
					t.Error("B should remain shared because its parent fans out")
				}
			},
		},
		{
			name: "keeps service shared when single parent references it twice",
			setup: func(c *Container) {
				s := &Service{ID: "S", Shared: true, Public: false}
				p := &Service{
					ID:             "P",
					Shared:         true,
					Dependencies:   []*Service{s},
					DependencyRefs: map[string]int{"S": 2},
				}
				c.Services["S"] = s
				c.Services["P"] = p
			},
			validate: func(t *testing.T, c *Container) {
				if !c.Services["S"].Shared {
					t.Error("S should remain shared: its parent instantiates it twice")
				}
			},
		},
		{
			name: "keeps leaf shared when intermediate chain parent is referenced twice",
			setup: func(c *Container) {
				// A (shared) --2 refs--> B (non-shared) --1 ref--> C (shared)
				cSvc := &Service{ID: "C", Shared: true, Public: false}
				b := &Service{
					ID:             "B",
					Shared:         false,
					Public:         false,
					Dependencies:   []*Service{cSvc},
					DependencyRefs: map[string]int{"C": 1},
				}
				a := &Service{
					ID:             "A",
					Shared:         true,
					Dependencies:   []*Service{b},
					DependencyRefs: map[string]int{"B": 2},
				}
				c.Services["C"] = cSvc
				c.Services["B"] = b
				c.Services["A"] = a
			},
			validate: func(t *testing.T, c *Container) {
				if !c.Services["C"].Shared {
					t.Error("C should remain shared: two B instances would each build it")
				}
			},
		},
		{
			name: "optimizes chain of private shared services recursively (A->B->C)",
			setup: func(c *Container) {
				// C <- B <- A
				cSvc := &Service{ID: "C", Shared: true, Public: false}
				b := &Service{ID: "B", Shared: true, Public: false, Dependencies: []*Service{cSvc}}
				a := &Service{ID: "A", Shared: true, Public: true, Dependencies: []*Service{b}}

				c.Services["C"] = cSvc
				c.Services["B"] = b
				c.Services["A"] = a
			},
			validate: func(t *testing.T, c *Container) {
				// C is used by B. B is Shared (initially). So C -> Non-Shared.
				// B is used by A. A is Shared. So B -> Non-Shared.
				// A is Public. A -> Shared.

				if c.Services["C"].Shared {
					t.Error("C should be non-shared")
				}
				if c.Services["B"].Shared {
					t.Error("B should be non-shared")
				}
				if !c.Services["A"].Shared {
					t.Error("A should remain shared")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container := NewContainer()
			tt.setup(container)
			(&sharedOptimizerPhase{}).Apply(nil, container)
			tt.validate(t, container)
		})
	}
}
