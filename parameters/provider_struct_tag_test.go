package parameters

import (
	"errors"
	"testing"
	"time"
)

type namedPort int

type nestedConfig struct {
	Timeout string `di-param:"timeout"`
}

type taggedConfig struct {
	DSN       string        `di-param:"dsn"`
	Port      namedPort     `di-param:"port"`
	Debug     bool          `di-param:"debug"`
	Weight    float64       `di-param:"weight"`
	Ratio     float32       `di-param:"ratio"`
	Wait      time.Duration `di-param:"wait"`
	When      time.Time     `di-param:"when"`
	Nested    nestedConfig
	Prefixed  nestedConfig              `di-param:"pref"`
	Values    map[string]any            `di-param:"vals"`
	NestedMap map[string]map[string]int `di-param:"m"`
}

func TestProviderStructTagLookup(t *testing.T) {
	when := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	cfg := &taggedConfig{
		DSN:      "postgres://localhost/app",
		Port:     8080,
		Debug:    true,
		Weight:   1.5,
		Ratio:    2.5,
		Wait:     30 * time.Second,
		When:     when,
		Nested:   nestedConfig{Timeout: "30s"},
		Prefixed: nestedConfig{Timeout: "10s"},
		Values: map[string]any{
			"mode":    "fast",
			"retries": 3,
			"nested": map[string]any{
				"depth": 2,
			},
		},
		NestedMap: map[string]map[string]int{
			"inner": {"count": 7},
		},
	}

	provider := NewProviderStructTag(cfg)
	tests := []struct {
		name string
		want any
	}{
		{"dsn", "postgres://localhost/app"},
		// Named types are normalized to their base kind at lookup time.
		{"port", int64(8080)},
		{"debug", true},
		{"weight", 1.5},
		{"ratio", float32(2.5)},
		// Exact time.Duration and time.Time are preserved.
		{"wait", 30 * time.Second},
		{"when", when},
		{"timeout", "30s"},
		{"pref.timeout", "10s"},
		{"vals.mode", "fast"},
		{"vals.retries", int64(3)},
		{"vals.nested.depth", int64(2)},
		{"m.inner.count", int64(7)},
	}
	for _, tt := range tests {
		got, err := provider.Lookup(tt.name)
		if err != nil {
			t.Fatalf("Lookup(%s): unexpected error %v", tt.name, err)
		}
		if got != tt.want {
			t.Fatalf("Lookup(%s): got %v (%T), want %v (%T)", tt.name, got, got, tt.want, tt.want)
		}
	}

	if _, err := provider.Lookup("missing"); !errors.Is(err, ErrParameterNotFound) {
		t.Fatalf("expected ErrParameterNotFound, got %v", err)
	}
}

func TestProviderStructTagNil(t *testing.T) {
	var provider *ProviderStructTag
	if _, err := provider.Lookup("dsn"); !errors.Is(err, ErrParameterNotFound) {
		t.Fatalf("expected ErrParameterNotFound, got %v", err)
	}
}
