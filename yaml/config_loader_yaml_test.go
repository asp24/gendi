package yaml

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/asp24/gendi/imprt"
	"github.com/asp24/gendi/srcloc"
)

type stubResolver struct {
	paths map[string][]string
}

func (r stubResolver) CanResolve(string) bool { return true }

func (r stubResolver) Resolve(_, importPath string) ([]string, error) {
	if paths, ok := r.paths[importPath]; ok {
		return paths, nil
	}
	return nil, os.ErrNotExist
}

func writeFile(t *testing.T, dir, name, contents string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

func TestLoadAllowsDiamondImports(t *testing.T) {
	dir := t.TempDir()

	rootPath := writeFile(t, dir, "root.yaml", `
imports:
  - path: b
  - path: c
parameters:
  a:
    type: string
    value: "A"
`)
	bPath := writeFile(t, dir, "b.yaml", `
imports:
  - path: d
parameters:
  b:
    type: string
    value: "B"
`)
	cPath := writeFile(t, dir, "c.yaml", `
imports:
  - path: d
parameters:
  c:
    type: string
    value: "C"
`)
	dPath := writeFile(t, dir, "d.yaml", `
parameters:
  d:
    type: string
    value: "D"
`)

	loader := NewConfigLoaderYaml(stubResolver{
		paths: map[string][]string{
			"b": {bPath},
			"c": {cPath},
			"d": {dPath},
		},
	}, NewParser())

	readCount := 0
	origRead := defaultOsReadFile
	defaultOsReadFile = func(path string) ([]byte, error) {
		readCount++
		return os.ReadFile(path)
	}
	defer func() { defaultOsReadFile = origRead }()

	cfg, err := loader.Load(rootPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got, want := len(cfg.Parameters), 4; got != want {
		t.Fatalf("expected %d parameters, got %d", want, got)
	}
	if readCount != 4 {
		t.Fatalf("expected each file read once, got %d reads", readCount)
	}
}

func TestExcludeBasicGlob(t *testing.T) {
	dir := t.TempDir()
	servicesDir := filepath.Join(dir, "services")
	if err := os.MkdirAll(servicesDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	writeFile(t, servicesDir, "app.yaml", `
parameters:
  app:
    type: string
    value: "app"
`)
	writeFile(t, servicesDir, "db.yaml", `
parameters:
  db:
    type: string
    value: "db"
`)
	writeFile(t, servicesDir, "test_helper.yaml", `
parameters:
  test_helper:
    type: string
    value: "test"
`)

	rootPath := writeFile(t, dir, "gendi.yaml", `
imports:
  - path: ./services/*.yaml
    exclude:
      - ./services/test_*.yaml
`)

	loader := NewConfigLoaderYaml(imprt.NewResolverCompositeDefault(), NewParser())
	cfg, err := loader.Load(rootPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if _, ok := cfg.Parameters["app"]; !ok {
		t.Error("expected app parameter to be loaded")
	}
	if _, ok := cfg.Parameters["db"]; !ok {
		t.Error("expected db parameter to be loaded")
	}
	if _, ok := cfg.Parameters["test_helper"]; ok {
		t.Error("expected test_helper parameter to be excluded")
	}
}

func TestExcludeMultiplePatterns(t *testing.T) {
	dir := t.TempDir()
	servicesDir := filepath.Join(dir, "services")
	internalDir := filepath.Join(servicesDir, "internal")
	if err := os.MkdirAll(internalDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	writeFile(t, servicesDir, "app.yaml", `
parameters:
  app:
    type: string
    value: "app"
`)
	writeFile(t, servicesDir, "test_app.yaml", `
parameters:
  test_app:
    type: string
    value: "test"
`)
	writeFile(t, internalDir, "debug.yaml", `
parameters:
  debug:
    type: string
    value: "debug"
`)

	rootPath := writeFile(t, dir, "gendi.yaml", `
imports:
  - path: ./services/**/*.yaml
    exclude:
      - ./services/test_*.yaml
      - ./services/internal/*.yaml
`)

	loader := NewConfigLoaderYaml(imprt.NewResolverCompositeDefault(), NewParser())
	cfg, err := loader.Load(rootPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if _, ok := cfg.Parameters["app"]; !ok {
		t.Error("expected app parameter to be loaded")
	}
	if _, ok := cfg.Parameters["test_app"]; ok {
		t.Error("expected test_app parameter to be excluded")
	}
	if _, ok := cfg.Parameters["debug"]; ok {
		t.Error("expected debug parameter to be excluded")
	}
}

func TestExcludeAllFiles(t *testing.T) {
	dir := t.TempDir()
	servicesDir := filepath.Join(dir, "services")
	if err := os.MkdirAll(servicesDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	writeFile(t, servicesDir, "test1.yaml", `
parameters:
  test1:
    type: string
    value: "test1"
`)
	writeFile(t, servicesDir, "test2.yaml", `
parameters:
  test2:
    type: string
    value: "test2"
`)

	rootPath := writeFile(t, dir, "gendi.yaml", `
imports:
  - path: ./services/*.yaml
    exclude:
      - ./services/*.yaml
`)

	loader := NewConfigLoaderYaml(imprt.NewResolverCompositeDefault(), NewParser())
	cfg, err := loader.Load(rootPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(cfg.Parameters) != 0 {
		t.Errorf("expected no parameters, got %d", len(cfg.Parameters))
	}
}

func TestExcludeNonGlobImport(t *testing.T) {
	dir := t.TempDir()

	specificPath := writeFile(t, dir, "specific.yaml", `
parameters:
  specific:
    type: string
    value: "specific"
`)

	rootPath := writeFile(t, dir, "gendi.yaml", `
imports:
  - path: specific
    exclude:
      - ./specific.yaml
`)

	loader := NewConfigLoaderYaml(stubResolver{
		paths: map[string][]string{
			"specific": {specificPath},
		},
	}, NewParser())

	cfg, err := loader.Load(rootPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(cfg.Parameters) != 0 {
		t.Errorf("expected no parameters (file excluded), got %d", len(cfg.Parameters))
	}
}

func TestExcludeInvalidPattern(t *testing.T) {
	dir := t.TempDir()
	servicesDir := filepath.Join(dir, "services")
	if err := os.MkdirAll(servicesDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	writeFile(t, servicesDir, "app.yaml", `
parameters:
  app:
    type: string
    value: "app"
`)

	rootPath := writeFile(t, dir, "gendi.yaml", `
imports:
  - path: ./services/*.yaml
    exclude:
      - "[invalid"
`)

	loader := NewConfigLoaderYaml(imprt.NewResolverCompositeDefault(), NewParser())
	_, err := loader.Load(rootPath)
	if err == nil {
		t.Fatal("expected error for invalid exclusion pattern")
	}
}

func TestExcludeBackwardCompatibility(t *testing.T) {
	dir := t.TempDir()
	servicesDir := filepath.Join(dir, "services")
	if err := os.MkdirAll(servicesDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	writeFile(t, servicesDir, "app.yaml", `
parameters:
  app:
    type: string
    value: "app"
`)
	writeFile(t, servicesDir, "db.yaml", `
parameters:
  db:
    type: string
    value: "db"
`)

	// Test both scalar form and mapping form without exclude
	rootPath := writeFile(t, dir, "gendi.yaml", `
imports:
  - ./services/app.yaml
  - path: ./services/db.yaml
`)

	loader := NewConfigLoaderYaml(imprt.NewResolverCompositeDefault(), NewParser())
	cfg, err := loader.Load(rootPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if _, ok := cfg.Parameters["app"]; !ok {
		t.Error("expected app parameter to be loaded")
	}
	if _, ok := cfg.Parameters["db"]; !ok {
		t.Error("expected db parameter to be loaded")
	}
}

func TestExcludeNestedGlob(t *testing.T) {
	dir := t.TempDir()
	configDir := filepath.Join(dir, "config")
	prodDir := filepath.Join(configDir, "prod")
	devDir := filepath.Join(configDir, "dev")
	if err := os.MkdirAll(prodDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.MkdirAll(devDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	writeFile(t, prodDir, "database.yaml", `
parameters:
  prod_db:
    type: string
    value: "prod"
`)
	writeFile(t, devDir, "dev_database.yaml", `
parameters:
  dev_db:
    type: string
    value: "dev"
`)
	writeFile(t, configDir, "base.yaml", `
parameters:
  base:
    type: string
    value: "base"
`)

	rootPath := writeFile(t, dir, "gendi.yaml", `
imports:
  - path: ./config/**/*.yaml
    exclude:
      - ./config/**/dev_*.yaml
`)

	loader := NewConfigLoaderYaml(imprt.NewResolverCompositeDefault(), NewParser())
	cfg, err := loader.Load(rootPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if _, ok := cfg.Parameters["prod_db"]; !ok {
		t.Error("expected prod_db parameter to be loaded")
	}
	if _, ok := cfg.Parameters["base"]; !ok {
		t.Error("expected base parameter to be loaded")
	}
	if _, ok := cfg.Parameters["dev_db"]; ok {
		t.Error("expected dev_db parameter to be excluded")
	}
}

func TestExcludeAbsolutePath(t *testing.T) {
	dir := t.TempDir()
	servicesDir := filepath.Join(dir, "services")
	if err := os.MkdirAll(servicesDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	writeFile(t, servicesDir, "app.yaml", `
parameters:
  app:
    type: string
    value: "app"
`)
	testPath := writeFile(t, servicesDir, "test.yaml", `
parameters:
  test:
    type: string
    value: "test"
`)

	rootPath := writeFile(t, dir, "gendi.yaml", fmt.Sprintf(`
imports:
  - path: ./services/*.yaml
    exclude:
      - %s
`, testPath))

	loader := NewConfigLoaderYaml(imprt.NewResolverCompositeDefault(), NewParser())
	cfg, err := loader.Load(rootPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if _, ok := cfg.Parameters["app"]; !ok {
		t.Error("expected app parameter to be loaded")
	}
	if _, ok := cfg.Parameters["test"]; ok {
		t.Error("expected test parameter to be excluded")
	}
}

func TestLoad_UnmarshalError_HasLocation(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		wantLine int
		wantMsg  string
	}{
		{
			name:     "import missing path",
			yaml:     "imports:\n  - exclude:\n      - foo",
			wantLine: 2,
			wantMsg:  "import path is required",
		},
		{
			name:     "service wrong type",
			yaml:     "services:\n  my_svc:\n    - item",
			wantLine: 3,
			wantMsg:  "service must be a mapping or alias",
		},
		{
			name:     "tag missing name",
			yaml:     "services:\n  my_svc:\n    constructor:\n      func: fmt.Println\n    tags:\n      - \"\"",
			wantLine: 6,
			wantMsg:  "tag name is required",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := writeFile(t, dir, "gendi.yaml", tt.yaml)

			loader := NewConfigLoaderYaml(stubResolver{}, NewParser())
			_, err := loader.Load(path)
			if err == nil {
				t.Fatal("expected error")
			}

			var locErr *srcloc.Error
			if !errors.As(err, &locErr) {
				t.Fatalf("expected srcloc.Error, got %T: %v", err, err)
			}
			if locErr.Loc == nil {
				t.Fatal("expected non-nil location")
			}
			if locErr.Loc.Line != tt.wantLine {
				t.Errorf("Line = %d, want %d", locErr.Loc.Line, tt.wantLine)
			}
			if locErr.Message != tt.wantMsg {
				t.Errorf("Message = %q, want %q", locErr.Message, tt.wantMsg)
			}
			if locErr.Loc.File == "" {
				t.Error("expected non-empty File in location")
			}
		})
	}
}

func TestExcludeDirectory(t *testing.T) {
	dir := t.TempDir()
	servicesDir := filepath.Join(dir, "services")
	internalDir := filepath.Join(servicesDir, "internal")
	if err := os.MkdirAll(internalDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	writeFile(t, servicesDir, "app.yaml", `
parameters:
  app:
    type: string
    value: "app"
`)
	writeFile(t, internalDir, "secret.yaml", `
parameters:
  secret:
    type: string
    value: "secret"
`)
	writeFile(t, internalDir, "debug.yaml", `
parameters:
  debug:
    type: string
    value: "debug"
`)

	rootPath := writeFile(t, dir, "gendi.yaml", `
imports:
  - path: ./services/**/*.yaml
    exclude:
      - ./services/internal
`)

	loader := NewConfigLoaderYaml(imprt.NewResolverCompositeDefault(), NewParser())
	cfg, err := loader.Load(rootPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if _, ok := cfg.Parameters["app"]; !ok {
		t.Error("expected app parameter to be loaded")
	}
	if _, ok := cfg.Parameters["secret"]; ok {
		t.Error("expected secret parameter to be excluded")
	}
	if _, ok := cfg.Parameters["debug"]; ok {
		t.Error("expected debug parameter to be excluded")
	}
}
