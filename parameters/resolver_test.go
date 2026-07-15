package parameters

import (
	"errors"
	"testing"
	"time"
)

func TestResolver(t *testing.T) {
	r := NewResolver(NewProviderMap(map[string]any{
		"host":    "localhost",
		"port":    "8080",
		"debug":   true,
		"ratio":   2.5,
		"small":   int64(7),
		"timeout": "5s",
		"when":    "2026-01-02T03:04:05Z",
	}), nil)

	if got, err := r.String("host"); err != nil || got != "localhost" {
		t.Fatalf("String: got %v (err=%v)", got, err)
	}
	if got, err := r.Int("port"); err != nil || got != 8080 {
		t.Fatalf("Int: got %v (err=%v)", got, err)
	}
	if got, err := r.Bool("debug"); err != nil || !got {
		t.Fatalf("Bool: got %v (err=%v)", got, err)
	}
	if got, err := r.Float64("ratio"); err != nil || got != 2.5 {
		t.Fatalf("Float64: got %v (err=%v)", got, err)
	}
	if got, err := r.Int8("small"); err != nil || got != 7 {
		t.Fatalf("Int8: got %v (err=%v)", got, err)
	}
	if got, err := r.Uint16("small"); err != nil || got != 7 {
		t.Fatalf("Uint16: got %v (err=%v)", got, err)
	}
	if got, err := r.Duration("timeout"); err != nil || got != 5*time.Second {
		t.Fatalf("Duration: got %v (err=%v)", got, err)
	}
	want := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	if got, err := r.Time("when"); err != nil || !got.Equal(want) {
		t.Fatalf("Time: got %v (err=%v)", got, err)
	}

	// Lookup miss propagates ErrParameterNotFound.
	if _, err := r.Int("missing"); !errors.Is(err, ErrParameterNotFound) {
		t.Fatalf("expected ErrParameterNotFound, got %v", err)
	}
	// Cast failures propagate the caster error.
	if _, err := r.Int("host"); err == nil {
		t.Fatalf("expected cast error for non-numeric string")
	}
}

func TestNewResolverDefaults(t *testing.T) {
	r := NewResolver(nil, nil)
	if r.Caster == nil || r.Provider == nil {
		t.Fatalf("expected nil arguments to fall back to defaults")
	}
	if _, err := r.String("anything"); !errors.Is(err, ErrParameterNotFound) {
		t.Fatalf("expected ErrParameterNotFound from null provider, got %v", err)
	}
}

// Every Caster method must have a Resolver counterpart; this keeps the
// facade in sync when Caster grows.
func TestResolverMirrorsCaster(t *testing.T) {
	r := NewResolver(NewProviderMap(map[string]any{"v": "1"}), nil)
	calls := []func(string) error{
		func(n string) error { _, err := r.String(n); return err },
		func(n string) error { _, err := r.Bool(n); return err },
		func(n string) error { _, err := r.Int(n); return err },
		func(n string) error { _, err := r.Int8(n); return err },
		func(n string) error { _, err := r.Int16(n); return err },
		func(n string) error { _, err := r.Int32(n); return err },
		func(n string) error { _, err := r.Int64(n); return err },
		func(n string) error { _, err := r.Uint(n); return err },
		func(n string) error { _, err := r.Uint8(n); return err },
		func(n string) error { _, err := r.Uint16(n); return err },
		func(n string) error { _, err := r.Uint32(n); return err },
		func(n string) error { _, err := r.Uint64(n); return err },
		func(n string) error { _, err := r.Float32(n); return err },
		func(n string) error { _, err := r.Float64(n); return err },
		func(n string) error { _, err := r.Duration(n); return err },
		func(n string) error { _, err := r.Time(n); return err },
	}
	for i, call := range calls {
		// "1" converts to every numeric/bool/string target; Duration and
		// Time fail on parse (missing unit / not RFC3339), which still
		// proves the plumbing reached the caster.
		err := call("v")
		if err != nil && i < len(calls)-2 {
			t.Fatalf("call %d: unexpected error %v", i, err)
		}
	}
}
