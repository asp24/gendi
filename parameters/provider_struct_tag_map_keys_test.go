package parameters

import (
	"errors"
	"testing"
)

type paramKey string

type namedKeyConfig struct {
	Values map[paramKey]any `di-param:"vals"`
	ByID   map[int]string   `di-param:"ids"`
}

func TestProviderStructTagNamedMapKey(t *testing.T) {
	cfg := &namedKeyConfig{
		Values: map[paramKey]any{
			"mode": "fast",
			"nested": map[paramKey]any{
				"depth": 2,
			},
		},
		ByID: map[int]string{1: "one"},
	}

	provider := NewProviderStructTag(cfg)

	if got, err := provider.Lookup("vals.mode"); err != nil || got != "fast" {
		t.Fatalf("Lookup(vals.mode): expected fast, got %v (err=%v)", got, err)
	}
	if got, err := provider.Lookup("vals.nested.depth"); err != nil || got != int64(2) {
		t.Fatalf("Lookup(vals.nested.depth): expected 2, got %v (err=%v)", got, err)
	}

	// Non-string map keys cannot be addressed by a string path; the lookup
	// must miss instead of panicking.
	if _, err := provider.Lookup("ids.1"); !errors.Is(err, ErrParameterNotFound) {
		t.Fatalf("expected ErrParameterNotFound for non-string keyed map, got %v", err)
	}
}
