package yaml

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	di "github.com/asp24/gendi"
)

func getCurrentDir() string {
	_, filename, _, _ := runtime.Caller(0)

	return filepath.Dir(filename)
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
}

func TestInvalidImports(t *testing.T) {
	tests := []struct {
		name        string
		expectError string
	}{
		{
			name:        "import_nonexistent_file",
			expectError: "import.*not found",
		},
		{
			name:        "import_circular",
			expectError: "circular",
		},
		{
			name:        "import_invalid_yaml",
			expectError: "yaml",
		},
	}

	currentDir := getCurrentDir()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := filepath.Join(currentDir, "testdata", tt.name, "gendi.yaml")

			_, err := LoadConfig(configPath)

			if err == nil {
				t.Fatal("expected import error, got none")
			}

			matched, _ := regexp.MatchString("(?i)"+tt.expectError, err.Error())
			if !matched {
				t.Errorf("expected error matching %q, got: %v", tt.expectError, err)
			}
		})
	}
}

func TestLoadConfigAbsoluteImport(t *testing.T) {
	importPath := filepath.Join(getCurrentDir(), "testdata", "imports", "module.yaml")

	dir := t.TempDir()
	rootPath := filepath.Join(dir, "root.yaml")
	root := strings.TrimSpace(fmt.Sprintf(`
imports:
  - path: %q
`, importPath))
	writeTestFile(t,rootPath, root)

	cfg, err := LoadConfig(rootPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if _, ok := cfg.Services["module.service"]; !ok {
		t.Fatalf("expected service from absolute import to load")
	}
}

func TestLoadConfigImportGlobLocal(t *testing.T) {
	dir := t.TempDir()
	baseAPath := filepath.Join(dir, "base_a.yaml")
	baseBPath := filepath.Join(dir, "base_b.yaml")
	rootPath := filepath.Join(dir, "root.yaml")

	baseA := strings.TrimSpace(`
services:
  dupe:
    constructor:
      func: "example.NewA"
`)
	writeTestFile(t,baseAPath, baseA)

	baseB := strings.TrimSpace(`
services:
  dupe:
    constructor:
      func: "example.NewB"
  extra:
    constructor:
      func: "example.NewExtra"
`)
	writeTestFile(t,baseBPath, baseB)

	root := strings.TrimSpace(`
imports:
  - "./base_*.yaml"
`)
	writeTestFile(t,rootPath, root)

	cfg, err := LoadConfig(rootPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	dupe, ok := cfg.Services["dupe"]
	if !ok || dupe.Constructor.Func != "example.NewB" {
		t.Fatalf("expected dupe service to come from base_b")
	}
	if _, ok := cfg.Services["extra"]; !ok {
		t.Fatalf("expected extra service from base_b")
	}
}

func TestLoadConfigImportGlobLocalRecursive(t *testing.T) {
	dir := t.TempDir()
	configsDir := filepath.Join(dir, "configs")
	nestedDir := filepath.Join(configsDir, "nested")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatalf("create nested dir: %v", err)
	}

	baseAPath := filepath.Join(configsDir, "base_a.yaml")
	baseBPath := filepath.Join(nestedDir, "base_b.yaml")
	rootPath := filepath.Join(dir, "root.yaml")

	baseA := strings.TrimSpace(`
services:
  dupe:
    constructor:
      func: "example.NewA"
`)
	writeTestFile(t,baseAPath, baseA)

	baseB := strings.TrimSpace(`
services:
  dupe:
    constructor:
      func: "example.NewB"
  extra_recursive:
    constructor:
      func: "example.NewExtraRecursive"
`)
	writeTestFile(t,baseBPath, baseB)

	root := strings.TrimSpace(`
imports:
  - "./configs/**/*.yaml"
`)
	writeTestFile(t,rootPath, root)

	cfg, err := LoadConfig(rootPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	dupe, ok := cfg.Services["dupe"]
	if !ok || dupe.Constructor.Func != "example.NewB" {
		t.Fatalf("expected dupe service to come from nested base_b")
	}
	if _, ok = cfg.Services["extra_recursive"]; !ok {
		t.Fatalf("expected extra_recursive service from nested base_b")
	}
}

func TestLoadConfigAbsoluteImportGlob(t *testing.T) {
	importPath := filepath.Join(getCurrentDir(), "testdata", "imports", "*.yaml")

	dir := t.TempDir()
	rootPath := filepath.Join(dir, "root.yaml")
	root := strings.TrimSpace(fmt.Sprintf(`
imports:
  - path: %q
`, importPath))
	writeTestFile(t,rootPath, root)

	cfg, err := LoadConfig(rootPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if _, ok := cfg.Services["module.service"]; !ok {
		t.Fatalf("expected service from absolute glob import to load")
	}
	if _, ok := cfg.Services["module.extra"]; !ok {
		t.Fatalf("expected extra service from absolute glob import to load")
	}
}

func TestLoadConfigAbsoluteImportGlobRecursive(t *testing.T) {
	importPath := filepath.Join(getCurrentDir(), "testdata", "imports", "**", "*.yaml")

	dir := t.TempDir()
	rootPath := filepath.Join(dir, "root.yaml")
	root := strings.TrimSpace(fmt.Sprintf(`
imports:
  - path: %q
`, importPath))
	writeTestFile(t,rootPath, root)

	cfg, err := LoadConfig(rootPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if _, ok := cfg.Services["module.service"]; !ok {
		t.Fatalf("expected service from absolute recursive glob import to load")
	}
	if _, ok := cfg.Services["module.extra"]; !ok {
		t.Fatalf("expected extra service from absolute recursive glob import to load")
	}
	if _, ok := cfg.Services["module.nested"]; !ok {
		t.Fatalf("expected nested service from absolute recursive glob import to load")
	}
}

func TestLoadConfigServiceAlias(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join(getCurrentDir(), "testdata", "service_alias", "gendi.yaml"))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Services["alias"].Alias != "base" {
		t.Fatalf("expected alias to reference base")
	}
	if cfg.Services["alias_public"].Alias != "base" || !cfg.Services["alias_public"].Public {
		t.Fatalf("expected alias_public to be public and reference base")
	}
}

func TestLoadConfigNullArgument(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join(getCurrentDir(), "testdata", "null_argument", "gendi.yaml"))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	svc, ok := cfg.Services["svc"]
	if !ok || len(svc.Constructor.Args) != 1 {
		t.Fatalf("expected one constructor arg")
	}
	arg := svc.Constructor.Args[0]
	if arg.Kind != di.ArgLiteral {
		t.Fatalf("expected literal argument, got %v", arg.Kind)
	}
	if !arg.Literal.IsNull() {
		t.Fatalf("expected null literal, got %v", arg.Literal.Kind)
	}
}
