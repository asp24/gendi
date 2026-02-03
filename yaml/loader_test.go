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

// TestInvalidImports tests various import error scenarios.
// These tests remain in integration/ because they test the import resolution
// logic which involves file system operations and is separate from generation errors.
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

func TestLoadConfigModuleImport(t *testing.T) {
	modulePath := readModulePath(t)
	importPath := modulePath + "/generator/testdata/imports/module.yaml"

	dir, err := os.MkdirTemp(".", "config-import-")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})

	rootPath := filepath.Join(dir, "root.yaml")
	root := []byte(strings.TrimSpace(fmt.Sprintf(`
imports:
  - path: %q
`, importPath)))
	if err := os.WriteFile(rootPath, root, 0o644); err != nil {
		t.Fatalf("write root config: %v", err)
	}

	cfg, err := LoadConfig(rootPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if _, ok := cfg.Services["module.service"]; !ok {
		t.Fatalf("expected service from module import to load")
	}
}

func TestLoadConfigImportGlobLocal(t *testing.T) {
	dir := t.TempDir()
	baseAPath := filepath.Join(dir, "base_a.yaml")
	baseBPath := filepath.Join(dir, "base_b.yaml")
	rootPath := filepath.Join(dir, "root.yaml")

	baseA := []byte(strings.TrimSpace(`
services:
  dupe:
    constructor:
      func: "example.NewA"
`))
	if err := os.WriteFile(baseAPath, baseA, 0o644); err != nil {
		t.Fatalf("write base_a config: %v", err)
	}

	baseB := []byte(strings.TrimSpace(`
services:
  dupe:
    constructor:
      func: "example.NewB"
  extra:
    constructor:
      func: "example.NewExtra"
`))
	if err := os.WriteFile(baseBPath, baseB, 0o644); err != nil {
		t.Fatalf("write base_b config: %v", err)
	}

	root := []byte(strings.TrimSpace(`
imports:
  - "./base_*.yaml"
`))
	if err := os.WriteFile(rootPath, root, 0o644); err != nil {
		t.Fatalf("write root config: %v", err)
	}

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

	baseA := []byte(strings.TrimSpace(`
services:
  dupe:
    constructor:
      func: "example.NewA"
`))
	if err := os.WriteFile(baseAPath, baseA, 0o644); err != nil {
		t.Fatalf("write base_a config: %v", err)
	}

	baseB := []byte(strings.TrimSpace(`
services:
  dupe:
    constructor:
      func: "example.NewB"
  extra_recursive:
    constructor:
      func: "example.NewExtraRecursive"
`))
	if err := os.WriteFile(baseBPath, baseB, 0o644); err != nil {
		t.Fatalf("write base_b config: %v", err)
	}

	root := []byte(strings.TrimSpace(`
imports:
  - "./configs/**/*.yaml"
`))
	if err := os.WriteFile(rootPath, root, 0o644); err != nil {
		t.Fatalf("write root config: %v", err)
	}

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

func TestLoadConfigImportGlobModule(t *testing.T) {
	modulePath := readModulePath(t)
	importPath := modulePath + "/generator/testdata/imports/*.yaml"

	dir := t.TempDir()
	rootPath := filepath.Join(dir, "root.yaml")
	root := []byte(strings.TrimSpace(fmt.Sprintf(`
imports:
  - path: %q
`, importPath)))
	if err := os.WriteFile(rootPath, root, 0o644); err != nil {
		t.Fatalf("write root config: %v", err)
	}

	cfg, err := LoadConfig(rootPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if _, ok := cfg.Services["module.service"]; !ok {
		t.Fatalf("expected service from module glob import to load")
	}
	if _, ok := cfg.Services["module.extra"]; !ok {
		t.Fatalf("expected extra service from module glob import to load")
	}
}

func TestLoadConfigImportGlobModuleRecursive(t *testing.T) {
	modulePath := readModulePath(t)
	importPath := modulePath + "/generator/testdata/imports/**/*.yaml"

	dir := t.TempDir()
	rootPath := filepath.Join(dir, "root.yaml")
	root := []byte(strings.TrimSpace(fmt.Sprintf(`
imports:
  - path: %q
`, importPath)))
	if err := os.WriteFile(rootPath, root, 0o644); err != nil {
		t.Fatalf("write root config: %v", err)
	}

	cfg, err := LoadConfig(rootPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if _, ok := cfg.Services["module.service"]; !ok {
		t.Fatalf("expected service from module recursive glob import to load")
	}
	if _, ok := cfg.Services["module.extra"]; !ok {
		t.Fatalf("expected extra service from module recursive glob import to load")
	}
	if _, ok := cfg.Services["module.nested"]; !ok {
		t.Fatalf("expected nested service from module recursive glob import to load")
	}
}

func TestLoadConfigServiceAlias(t *testing.T) {
	dir := t.TempDir()
	rootPath := filepath.Join(dir, "root.yaml")
	root := []byte(strings.TrimSpace(`
services:
  base:
    constructor:
      func: "example.NewBase"
  alias: "@base"
  alias_public:
    alias: "base"
    public: true
`))
	if err := os.WriteFile(rootPath, root, 0o644); err != nil {
		t.Fatalf("write root config: %v", err)
	}

	cfg, err := LoadConfig(rootPath)
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
	dir := t.TempDir()
	rootPath := filepath.Join(dir, "root.yaml")
	root := []byte(strings.TrimSpace(`
services:
  svc:
    constructor:
      func: "example.NewB"
      args:
        - null
`))
	if err := os.WriteFile(rootPath, root, 0o644); err != nil {
		t.Fatalf("write root config: %v", err)
	}

	cfg, err := LoadConfig(rootPath)
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

func readModulePath(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("../go.mod")
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	t.Fatalf("module path not found in go.mod")
	return ""
}
