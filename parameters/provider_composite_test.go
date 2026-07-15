package parameters

import (
	"errors"
	"testing"
)

func TestParametersProviderComposite(t *testing.T) {
	first := NewProviderMap(map[string]any{
		"dsn": "postgres://fallback/app",
	})
	second := NewProviderMap(map[string]any{
		"dsn": "postgres://primary/app",
	})

	composite := NewProviderComposite(first, second)
	if got, err := composite.Lookup("dsn"); err != nil || got != "postgres://primary/app" {
		t.Fatalf("expected primary value, got %v (err=%v)", got, err)
	}

	missing := NewProviderComposite(NewProviderMap(nil))
	if _, err := missing.Lookup("dsn"); !errors.Is(err, ErrParameterNotFound) {
		t.Fatalf("expected ErrParameterNotFound, got %v", err)
	}
}
