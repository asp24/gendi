package ir

import (
	"testing"
)

func TestOptimizeShared(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(ctx *buildContext)
		validate func(t *testing.T, ctx *buildContext)
	}{
		{
			name: "optimizes shared service used by single shared parent",
			setup: func(ctx *buildContext) {
				s := &Service{ID: "S", Shared: true, Public: false}
				p := &Service{ID: "P", Shared: true, Dependencies: []*Service{s}}
				ctx.services["S"] = s
				ctx.services["P"] = p
			},
			validate: func(t *testing.T, ctx *buildContext) {
				if ctx.services["S"].Shared {
					t.Error("S should be non-shared")
				}
			},
		},
		{
			name: "does not optimize if parent is non-shared",
			setup: func(ctx *buildContext) {
				s := &Service{ID: "S", Shared: true, Public: false}
				p := &Service{ID: "P", Shared: false, Dependencies: []*Service{s}}
				ctx.services["S"] = s
				ctx.services["P"] = p
			},
			validate: func(t *testing.T, ctx *buildContext) {
				if !ctx.services["S"].Shared {
					t.Error("S should remain shared")
				}
			},
		},
		{
			name: "does not optimize if used by multiple services",
			setup: func(ctx *buildContext) {
				s := &Service{ID: "S", Shared: true, Public: false}
				p1 := &Service{ID: "P1", Shared: true, Dependencies: []*Service{s}}
				p2 := &Service{ID: "P2", Shared: true, Dependencies: []*Service{s}}
				ctx.services["S"] = s
				ctx.services["P1"] = p1
				ctx.services["P2"] = p2
			},
			validate: func(t *testing.T, ctx *buildContext) {
				if !ctx.services["S"].Shared {
					t.Error("S should remain shared")
				}
			},
		},
		{
			name: "does not optimize public service",
			setup: func(ctx *buildContext) {
				s := &Service{ID: "S", Shared: true, Public: true}
				p := &Service{ID: "P", Shared: true, Dependencies: []*Service{s}}
				ctx.services["S"] = s
				ctx.services["P"] = p
			},
			validate: func(t *testing.T, ctx *buildContext) {
				if !ctx.services["S"].Shared {
					t.Error("S should remain shared")
				}
			},
		},
		{
			name: "does not optimize tagged service",
			setup: func(ctx *buildContext) {
				s := &Service{
					ID:     "S",
					Shared: true,
					Public: false,
					Tags:   []*ServiceTag{{Tag: &Tag{Public: true}}},
				}
				p := &Service{ID: "P", Shared: true, Dependencies: []*Service{s}}
				ctx.services["S"] = s
				ctx.services["P"] = p
			},
			validate: func(t *testing.T, ctx *buildContext) {
				if !ctx.services["S"].Shared {
					t.Error("S should remain shared")
				}
			},
		},
		{
			name: "does not optimize alias",
			setup: func(ctx *buildContext) {
				target := &Service{ID: "T", Shared: true}
				s := &Service{
					ID:     "S",
					Shared: true,
					Public: false,
					Alias:  target,
				}
				p := &Service{ID: "P", Shared: true, Dependencies: []*Service{s}}
				ctx.services["S"] = s
				ctx.services["T"] = target
				ctx.services["P"] = p
			},
			validate: func(t *testing.T, ctx *buildContext) {
				if !ctx.services["S"].Shared {
					t.Error("S should remain shared")
				}
			},
		},
		{
			name: "optimizes chain of private shared services recursively (A->B->C)",
			setup: func(ctx *buildContext) {
				// C <- B <- A
				c := &Service{ID: "C", Shared: true, Public: false}
				b := &Service{ID: "B", Shared: true, Public: false, Dependencies: []*Service{c}}
				a := &Service{ID: "A", Shared: true, Public: true, Dependencies: []*Service{b}}

				ctx.services["C"] = c
				ctx.services["B"] = b
				ctx.services["A"] = a
			},
			validate: func(t *testing.T, ctx *buildContext) {
				// C is used by B. B is Shared (initially). So C -> Non-Shared.
				// B is used by A. A is Shared. So B -> Non-Shared.
				// A is Public. A -> Shared.

				if ctx.services["C"].Shared {
					t.Error("C should be non-shared")
				}
				if ctx.services["B"].Shared {
					t.Error("B should be non-shared")
				}
				if !ctx.services["A"].Shared {
					t.Error("A should remain shared")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &buildContext{
				services: make(map[string]*Service),
			}
			tt.setup(ctx)
			(&sharedOptimizer{}).resolve(ctx)
			tt.validate(t, ctx)
		})
	}
}
