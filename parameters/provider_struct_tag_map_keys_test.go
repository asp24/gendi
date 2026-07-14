package parameters

import "testing"

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

	if !provider.Has("vals.mode") {
		t.Fatalf("expected Has(vals.mode) to be true")
	}
	if got, err := provider.GetString("vals.mode"); err != nil || got != "fast" {
		t.Fatalf("GetString: expected fast, got %v (err=%v)", got, err)
	}
	if got, err := provider.GetInt("vals.nested.depth"); err != nil || got != 2 {
		t.Fatalf("nested GetInt: expected 2, got %v (err=%v)", got, err)
	}

	// Non-string map keys cannot be addressed by a string path; the lookup
	// must miss instead of panicking.
	if provider.Has("ids.1") {
		t.Fatalf("expected Has(ids.1) to be false for non-string keyed map")
	}
}
