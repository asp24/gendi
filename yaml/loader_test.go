package yaml

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	di "github.com/gendi-org/gendi"
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

			_, err := LoadConfig(configPath, boundaryFor(t, configPath))

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

// writeModuleImportsFixture populates root with a go.mod (module
// example.com/app) and an imports/ tree used by the own-module import tests:
// files inside the module are reachable through the module's own import path,
// "example.com/app/imports/...".
func writeModuleImportsFixture(t *testing.T, root string) {
	t.Helper()
	writeTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n")
	nested := filepath.Join(root, "imports", "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeTestFile(t, filepath.Join(root, "imports", "module.yaml"), strings.TrimSpace(`
services:
  module.service:
    constructor:
      func: "example.NewModuleService"
`))
	writeTestFile(t, filepath.Join(root, "imports", "module_extra.yaml"), strings.TrimSpace(`
services:
  module.extra:
    constructor:
      func: "example.NewModuleExtra"
`))
	writeTestFile(t, filepath.Join(nested, "module_nested.yaml"), strings.TrimSpace(`
services:
  module.nested:
    constructor:
      func: "example.NewModuleNested"
`))
}

// Absolute filesystem paths are not allowed in imports — even ones pointing
// inside the module. Files in the module are addressed relatively or through
// the module's own import path.
func TestLoadConfigRejectsAbsoluteImport(t *testing.T) {
	dir := t.TempDir()
	writeModuleImportsFixture(t, dir)

	rootPath := filepath.Join(dir, "root.yaml")
	writeTestFile(t, rootPath, strings.TrimSpace(fmt.Sprintf(`
imports:
  - path: %q
`, filepath.Join(dir, "imports", "module.yaml"))))

	_, err := LoadConfig(rootPath, boundaryFor(t, rootPath))
	if err == nil {
		t.Fatal("expected error for absolute import path")
	}
	if !strings.Contains(err.Error(), "absolute") {
		t.Fatalf("error should mention absolute paths, got: %v", err)
	}
}

// A config file inside the module is importable through the module's own
// import path, giving a location-independent anchor at the module root.
func TestLoadConfigOwnModuleImport(t *testing.T) {
	dir := t.TempDir()
	writeModuleImportsFixture(t, dir)

	rootPath := filepath.Join(dir, "root.yaml")
	writeTestFile(t, rootPath, strings.TrimSpace(`
imports:
  - path: example.com/app/imports/module.yaml
`))

	cfg, err := LoadConfig(rootPath, boundaryFor(t, rootPath))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if _, ok := cfg.Services["module.service"]; !ok {
		t.Fatalf("expected service from own-module import to load")
	}
}

// An import that escapes the module root via ".." is rejected.
func TestLoadConfigRejectsEscapingImport(t *testing.T) {
	outer := t.TempDir()
	writeTestFile(t, filepath.Join(outer, "secret.yaml"), strings.TrimSpace(`
parameters:
  secret: "leaked"
`))

	moduleRoot := filepath.Join(outer, "module")
	if err := os.MkdirAll(moduleRoot, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeTestFile(t, filepath.Join(moduleRoot, "go.mod"), "module example.com/app\n")

	rootPath := filepath.Join(moduleRoot, "root.yaml")
	writeTestFile(t, rootPath, strings.TrimSpace(`
imports:
  - path: ../secret.yaml
`))

	if _, err := LoadConfig(rootPath, boundaryFor(t, rootPath)); err == nil {
		t.Fatal("expected error for import escaping the module root")
	}
}

// A relative import whose first segment merely contains a dot (so it looks
// module-shaped) but which uses ".." to climb out of the module root must be
// rejected, exactly like a plain "../" escape.
func TestLoadConfigRejectsDottedSegmentEscape(t *testing.T) {
	outer := t.TempDir()
	writeTestFile(t, filepath.Join(outer, "secret.yaml"), strings.TrimSpace(`
parameters:
  secret: "leaked"
`))

	moduleRoot := filepath.Join(outer, "module")
	if err := os.MkdirAll(moduleRoot, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeTestFile(t, filepath.Join(moduleRoot, "go.mod"), "module example.com/app\n")

	rootPath := filepath.Join(moduleRoot, "root.yaml")
	writeTestFile(t, rootPath, strings.TrimSpace(`
imports:
  - path: assets.d/../../secret.yaml
`))

	if _, err := LoadConfig(rootPath, boundaryFor(t, rootPath)); err == nil {
		t.Fatal("expected error for dotted-segment import escaping the module root")
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
	writeTestFile(t, baseAPath, baseA)

	baseB := strings.TrimSpace(`
services:
  dupe:
    constructor:
      func: "example.NewB"
  extra:
    constructor:
      func: "example.NewExtra"
`)
	writeTestFile(t, baseBPath, baseB)

	root := strings.TrimSpace(`
imports:
  - "./base_*.yaml"
`)
	writeTestFile(t, rootPath, root)

	cfg, err := LoadConfig(rootPath, boundaryFor(t, rootPath))
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
	writeTestFile(t, baseAPath, baseA)

	baseB := strings.TrimSpace(`
services:
  dupe:
    constructor:
      func: "example.NewB"
  extra_recursive:
    constructor:
      func: "example.NewExtraRecursive"
`)
	writeTestFile(t, baseBPath, baseB)

	root := strings.TrimSpace(`
imports:
  - "./configs/**/*.yaml"
`)
	writeTestFile(t, rootPath, root)

	cfg, err := LoadConfig(rootPath, boundaryFor(t, rootPath))
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

func TestLoadConfigOwnModuleImportGlob(t *testing.T) {
	dir := t.TempDir()
	writeModuleImportsFixture(t, dir)

	rootPath := filepath.Join(dir, "root.yaml")
	writeTestFile(t, rootPath, strings.TrimSpace(`
imports:
  - path: example.com/app/imports/*.yaml
`))

	cfg, err := LoadConfig(rootPath, boundaryFor(t, rootPath))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if _, ok := cfg.Services["module.service"]; !ok {
		t.Fatalf("expected service from own-module glob import to load")
	}
	if _, ok := cfg.Services["module.extra"]; !ok {
		t.Fatalf("expected extra service from own-module glob import to load")
	}
}

func TestLoadConfigOwnModuleImportGlobRecursive(t *testing.T) {
	dir := t.TempDir()
	writeModuleImportsFixture(t, dir)

	rootPath := filepath.Join(dir, "root.yaml")
	writeTestFile(t, rootPath, strings.TrimSpace(`
imports:
  - path: example.com/app/imports/**/*.yaml
`))

	cfg, err := LoadConfig(rootPath, boundaryFor(t, rootPath))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if _, ok := cfg.Services["module.service"]; !ok {
		t.Fatalf("expected service from own-module recursive glob import to load")
	}
	if _, ok := cfg.Services["module.extra"]; !ok {
		t.Fatalf("expected extra service from own-module recursive glob import to load")
	}
	if _, ok := cfg.Services["module.nested"]; !ok {
		t.Fatalf("expected nested service from own-module recursive glob import to load")
	}
}

func TestLoadConfigServiceAlias(t *testing.T) {
	configPath := filepath.Join(getCurrentDir(), "testdata", "service_alias", "gendi.yaml")
	cfg, err := LoadConfig(configPath, boundaryFor(t, configPath))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Services["alias"].Alias != "base" {
		t.Fatalf("expected alias to reference base")
	}
	if cfg.Services["alias"].Shared {
		t.Fatal("expected alias shorthand to have no shared setting")
	}
	if cfg.Services["alias_public"].Alias != "base" || !cfg.Services["alias_public"].Public {
		t.Fatalf("expected alias_public to be public and reference base")
	}
	if cfg.Services["alias_public"].Shared {
		t.Fatal("expected expanded alias to have no shared setting")
	}
}

func TestLoadConfigDecoratorDefaultsToShared(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gendi.yaml")
	writeTestFile(t, path, strings.TrimSpace(`
services:
  base:
    constructor:
      func: "app.NewBase"
  decorator:
    decorates: base
    constructor:
      func: "app.NewDecorator"
      args: ["@.inner"]
`))

	cfg, err := LoadConfig(path, boundaryFor(t, path))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !cfg.Services["base"].Shared {
		t.Fatal("expected base to default to shared")
	}
	if !cfg.Services["decorator"].Shared {
		t.Fatal("expected decorator to default to shared")
	}
}

func TestLoadConfigNullArgument(t *testing.T) {
	configPath := filepath.Join(getCurrentDir(), "testdata", "null_argument", "gendi.yaml")
	cfg, err := LoadConfig(configPath, boundaryFor(t, configPath))
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
