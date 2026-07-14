package ir

import "testing"

func TestCycleErrorDeterministic(t *testing.T) {
	build := func() *Container {
		c := NewContainer()
		a := &Service{ID: "a", Public: true}
		b := &Service{ID: "b"}
		y := &Service{ID: "y"}
		z := &Service{ID: "z"}
		a.Dependencies = []*Service{b}
		b.Dependencies = []*Service{a}
		y.Dependencies = []*Service{z}
		z.Dependencies = []*Service{y}
		c.Services["a"] = a
		c.Services["b"] = b
		c.Services["y"] = y
		c.Services["z"] = z
		return c
	}

	v := &validatorPhase{}
	first := v.detectCycles(build()).Error()
	for i := 0; i < 100; i++ {
		if got := v.detectCycles(build()).Error(); got != first {
			t.Fatalf("cycle error is nondeterministic: %q vs %q", first, got)
		}
	}
}
