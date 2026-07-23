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
	return LoadConfig(path, dir, dir)
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

func TestMappingParameterRejected(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{
			name: "mapping with several keys",
			yaml: `
parameters:
  port:
    type: int
    value: 8080

services:
  app:
    constructor:
      func: "test.NewApp"
`,
			wantErr: "value must be a plain scalar, got a mapping",
		},
		{
			name: "mapping with single key",
			yaml: `
parameters:
  port:
    value: 7

services:
  app:
    constructor:
      func: "test.NewApp"
`,
			wantErr: "value must be a plain scalar, got a mapping",
		},
		{
			name: "null value",
			yaml: `
parameters:
  port: ~

services:
  app:
    constructor:
      func: "test.NewApp"
`,
			wantErr: "null value is not supported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := loadConfigString(t, tt.yaml)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}
