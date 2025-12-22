package parameters

import (
	"errors"
	"testing"
)

func TestParametersProviderMap(t *testing.T) {
	provider := NewProviderMap(map[string]interface{}{
		"port":   8080,
		"debug":  true,
		"dsn":    "postgres://localhost/app",
		"weight": 1.5,
	})

	if got, err := provider.GetInt("port"); err != nil || got != 8080 {
		t.Fatalf("GetInt: expected 8080, got %v (err=%v)", got, err)
	}
	if got, err := provider.GetBool("debug"); err != nil || !got {
		t.Fatalf("GetBool: expected true, got %v (err=%v)", got, err)
	}
	if got, err := provider.GetString("dsn"); err != nil || got == "" {
		t.Fatalf("GetString: expected non-empty, got %v (err=%v)", got, err)
	}
	if got, err := provider.GetFloat("weight"); err != nil || got != 1.5 {
		t.Fatalf("GetFloat: expected 1.5, got %v (err=%v)", got, err)
	}

	if _, err := provider.GetString("missing"); !errors.Is(err, ErrParameterNotFound) {
		t.Fatalf("expected ErrParameterNotFound, got %v", err)
	}
	if _, err := provider.GetInt("dsn"); err == nil {
		t.Fatalf("expected type error for GetInt on string")
	}
}

func TestParametersProviderComposite(t *testing.T) {
	first := NewProviderMap(map[string]interface{}{
		"dsn": "postgres://primary/app",
	})
	second := NewProviderMap(map[string]interface{}{
		"dsn": "postgres://fallback/app",
	})

	composite := NewProviderComposite(first, second)
	if got, err := composite.GetString("dsn"); err != nil || got != "postgres://primary/app" {
		t.Fatalf("expected primary value, got %v (err=%v)", got, err)
	}

	missing := NewProviderComposite(NewProviderMap(nil))
	if _, err := missing.GetString("dsn"); !errors.Is(err, ErrParameterNotFound) {
		t.Fatalf("expected ErrParameterNotFound, got %v", err)
	}
}
