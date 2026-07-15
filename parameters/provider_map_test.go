package parameters

import (
	"errors"
	"testing"
)

func TestParametersProviderMap(t *testing.T) {
	provider := NewProviderMap(map[string]any{
		"port": 8080,
		"dsn":  "postgres://localhost/app",
	})

	if got, err := provider.Lookup("port"); err != nil || got != 8080 {
		t.Fatalf("Lookup(port): expected 8080, got %v (err=%v)", got, err)
	}
	if got, err := provider.Lookup("dsn"); err != nil || got != "postgres://localhost/app" {
		t.Fatalf("Lookup(dsn): unexpected %v (err=%v)", got, err)
	}
	if _, err := provider.Lookup("missing"); !errors.Is(err, ErrParameterNotFound) {
		t.Fatalf("expected ErrParameterNotFound, got %v", err)
	}
}

func TestParametersProviderMapNil(t *testing.T) {
	if _, err := NewProviderMap(nil).Lookup("x"); !errors.Is(err, ErrParameterNotFound) {
		t.Fatalf("expected ErrParameterNotFound, got %v", err)
	}
}
