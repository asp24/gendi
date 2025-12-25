package parameters

import (
	"errors"
	"testing"
	"time"
)

type nestedConfig struct {
	Timeout string `di-param:"timeout"`
}

type taggedConfig struct {
	DSN       string  `di-param:"dsn"`
	Port      int     `di-param:"port"`
	Debug     bool    `di-param:"debug"`
	Weight    float64 `di-param:"weight"`
	Nested    nestedConfig
	Prefixed  nestedConfig              `di-param:"pref"`
	Values    map[string]any            `di-param:"vals"`
	NestedMap map[string]map[string]int `di-param:"m"`
}

func TestProviderStructTag(t *testing.T) {
	cfg := &taggedConfig{
		DSN:      "postgres://localhost/app",
		Port:     8080,
		Debug:    true,
		Weight:   1.5,
		Nested:   nestedConfig{Timeout: "30s"},
		Prefixed: nestedConfig{Timeout: "10s"},
		Values: map[string]any{
			"mode":    "fast",
			"retries": 3,
			"enabled": true,
			"ratio":   2.5,
			"nested": map[string]any{
				"depth": 2,
			},
		},
		NestedMap: map[string]map[string]int{
			"inner": {
				"count": 7,
			},
		},
	}

	provider := NewProviderStructTag(cfg)
	if got, err := provider.GetString("dsn"); err != nil || got == "" {
		t.Fatalf("GetString: expected value, got %v (err=%v)", got, err)
	}
	if got, err := provider.GetInt("port"); err != nil || got != 8080 {
		t.Fatalf("GetInt: expected 8080, got %v (err=%v)", got, err)
	}
	if got, err := provider.GetBool("debug"); err != nil || !got {
		t.Fatalf("GetBool: expected true, got %v (err=%v)", got, err)
	}
	if got, err := provider.GetFloat("weight"); err != nil || got != 1.5 {
		t.Fatalf("GetFloat: expected 1.5, got %v (err=%v)", got, err)
	}
	if got, err := provider.GetDuration("timeout"); err != nil || got != time.Second*30 {
		t.Fatalf("nested GetDuration: expected 30s, got %v (err=%v)", got, err)
	}
	if got, err := provider.GetDuration("pref.timeout"); err != nil || got != time.Second*10 {
		t.Fatalf("prefixed GetDuration: expected 10s, got %v (err=%v)", got, err)
	}
	if got, err := provider.GetString("vals.mode"); err != nil || got != "fast" {
		t.Fatalf("map GetString: expected fast, got %v (err=%v)", got, err)
	}
	if got, err := provider.GetInt("vals.retries"); err != nil || got != 3 {
		t.Fatalf("map GetInt: expected 3, got %v (err=%v)", got, err)
	}
	if got, err := provider.GetBool("vals.enabled"); err != nil || !got {
		t.Fatalf("map GetBool: expected true, got %v (err=%v)", got, err)
	}
	if got, err := provider.GetFloat("vals.ratio"); err != nil || got != 2.5 {
		t.Fatalf("map GetFloat: expected 2.5, got %v (err=%v)", got, err)
	}
	if got, err := provider.GetInt("vals.nested.depth"); err != nil || got != 2 {
		t.Fatalf("nested map GetInt: expected 2, got %v (err=%v)", got, err)
	}
	if got, err := provider.GetInt("m.inner.count"); err != nil || got != 7 {
		t.Fatalf("typed map GetInt: expected 7, got %v (err=%v)", got, err)
	}

	if !provider.Has("dsn") || !provider.Has("vals.mode") {
		t.Fatalf("expected Has to return true for existing params")
	}
	if provider.Has("missing") {
		t.Fatalf("expected Has to return false for missing param")
	}
	if _, err := provider.GetString("missing"); !errors.Is(err, ErrParameterNotFound) {
		t.Fatalf("expected ErrParameterNotFound, got %v", err)
	}
}

func TestProviderStructTagNil(t *testing.T) {
	var provider *ProviderStructTag
	if _, err := provider.GetString("dsn"); !errors.Is(err, ErrParameterNotFound) {
		t.Fatalf("expected ErrParameterNotFound, got %v", err)
	}
}
