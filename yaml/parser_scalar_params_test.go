package yaml

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	di "github.com/gendi-org/gendi"
)

func loadConfigString(t *testing.T, content string) (*di.Config, error) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "gendi.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return LoadConfig(path)
}

func TestScalarParameters(t *testing.T) {
	cfg, err := loadConfigString(t, `
parameters:
  port: 8080
  host: localhost
  ratio: 1.5
  debug: true
  timeout: 5s

services:
  app:
    constructor:
      func: "test.NewApp"
      args: ["%host%"]
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tests := []struct {
		name string
		kind di.LiteralKind
	}{
		{"port", di.LiteralInt},
		{"host", di.LiteralString},
		{"ratio", di.LiteralFloat},
		{"debug", di.LiteralBool},
		{"timeout", di.LiteralString},
	}
	for _, tt := range tests {
		p, ok := cfg.Parameters[tt.name]
		if !ok {
			t.Fatalf("parameter %q missing", tt.name)
		}
		if p.Value.Kind != tt.kind {
			t.Fatalf("parameter %q: kind %v, want %v", tt.name, p.Value.Kind, tt.kind)
		}
	}
}

func TestMappingParameterDeprecatedFormStillLoads(t *testing.T) {
	cfg, err := loadConfigString(t, `
parameters:
  port:
    type: int
    value: 8080

services:
  app:
    constructor:
      func: "test.NewApp"
`)
	if err != nil {
		t.Fatalf("deprecated form must still load, got %v", err)
	}
	if got := cfg.Parameters["port"].Value; got.Kind != di.LiteralInt || got.Int() != 8080 {
		t.Fatalf("expected int 8080 from value field, got %+v", got)
	}
}

func TestMappingParameterUnknownKeyRejected(t *testing.T) {
	_, err := loadConfigString(t, `
parameters:
  port:
    typo: silently-ignored
    value: 7

services:
  app:
    constructor:
      func: "test.NewApp"
`)
	if err == nil || !strings.Contains(err.Error(), `unsupported key "typo"`) {
		t.Fatalf("expected unsupported key error, got %v", err)
	}
}

func TestMappingParameterWithoutValueRejected(t *testing.T) {
	_, err := loadConfigString(t, `
parameters:
  port:
    type: int

services:
  app:
    constructor:
      func: "test.NewApp"
`)
	if err == nil || !strings.Contains(err.Error(), "requires a value") {
		t.Fatalf("expected missing value error, got %v", err)
	}
}

func TestNullParameterRejected(t *testing.T) {
	_, err := loadConfigString(t, `
parameters:
  port: ~

services:
  app:
    constructor:
      func: "test.NewApp"
`)
	if err == nil || !strings.Contains(err.Error(), "null value is not supported") {
		t.Fatalf("expected null rejection, got %v", err)
	}
}
