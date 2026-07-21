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

func loadConfigWithDefaultBoundary(t *testing.T, path string) (*di.Config, error) {
	t.Helper()
	boundary := boundaryFor(t, path)
	return LoadConfig(path, boundary, boundary)
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

			_, err := loadConfigWithDefaultBoundary(t, configPath)

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

// A config imported through a symlink anchors its own relative imports (and
// $this) at the symlink's directory — the addressed location — not at the
// real file's directory. This is the overlay pattern: one template symlinked
// into several env directories picks up each directory's local files.
func TestLoadConfigSymlinkedImportAnchorsAtSpelledDir(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{"env", "envs"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}
	writeTestFile(t, filepath.Join(dir, "envs", "prod.yaml"), strings.TrimSpace(`
imports:
  - ./common.yaml
`))
	writeTestFile(t, filepath.Join(dir, "envs", "common.yaml"), strings.TrimSpace(`
parameters:
  which: "envs"
`))
	writeTestFile(t, filepath.Join(dir, "env", "common.yaml"), strings.TrimSpace(`
parameters:
  which: "env"
`))
	if err := os.Symlink(filepath.Join("..", "envs", "prod.yaml"), filepath.Join(dir, "env", "gendi.yaml")); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	rootPath := filepath.Join(dir, "root.yaml")
	writeTestFile(t, rootPath, strings.TrimSpace(`
imports:
  - ./env/gendi.yaml
`))

	cfg, err := loadConfigWithDefaultBoundary(t, rootPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if got := cfg.Parameters["which"].Value.String(); got != "env" {
		t.Fatalf("symlinked config must anchor at the symlink's directory: which = %q, want %q", got, "env")
	}
}

func TestLoadConfigLoadsEachSymlinkAliasInItsAddressedContext(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n")
	writeTestFile(t, filepath.Join(root, "template.yaml"), strings.TrimSpace(`
imports:
  - ./common.yaml
services:
  contextual:
    type: $this.Service
`))

	for _, env := range []string{"dev", "prod"} {
		dir := filepath.Join(root, "env", env)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", env, err)
		}
		writeTestFile(t, filepath.Join(dir, "common.yaml"), fmt.Sprintf("parameters: {%s: loaded}", env))
		if err := os.Symlink(filepath.Join("..", "..", "template.yaml"), filepath.Join(dir, "gendi.yaml")); err != nil {
			t.Fatalf("symlink %s: %v", env, err)
		}
	}

	rootPath := filepath.Join(root, "gendi.yaml")
	writeTestFile(t, rootPath, "imports: [./env/**/gendi.yaml]")

	cfg, err := loadConfigWithDefaultBoundary(t, rootPath)
	if err != nil {
		t.Fatalf("load aliases: %v", err)
	}
	for _, name := range []string{"dev", "prod"} {
		if _, ok := cfg.Parameters[name]; !ok {
			t.Errorf("expected parameter from %s alias context", name)
		}
	}
	if got := cfg.Services["contextual"].Type; got != "example.com/app/env/prod.Service" {
		t.Fatalf("$this resolved to %q, want last alias context", got)
	}
}

func TestLoadConfigRejectsRootSymlinkOutsideBoundaryBeforeRead(t *testing.T) {
	outer := t.TempDir()
	external := filepath.Join(outer, "external.yaml")
	writeTestFile(t, external, "parameters: {secret: leaked}")

	moduleRoot := filepath.Join(outer, "module")
	if err := os.MkdirAll(moduleRoot, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeTestFile(t, filepath.Join(moduleRoot, "go.mod"), "module example.com/app\n")
	rootPath := filepath.Join(moduleRoot, "gendi.yaml")
	if err := os.Symlink(external, rootPath); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	boundary, err := DefaultBoundary(rootPath)
	if err != nil {
		t.Fatalf("default boundary: %v", err)
	}

	readCount := 0
	origRead := defaultOsReadFile
	defaultOsReadFile = func(path string) ([]byte, error) {
		readCount++
		return os.ReadFile(path)
	}
	defer func() { defaultOsReadFile = origRead }()

	_, err = LoadConfig(rootPath, boundary, boundary)
	if err == nil || !strings.Contains(err.Error(), "outside boundary") {
		t.Fatalf("expected root confinement error, got %v", err)
	}
	if readCount != 0 {
		t.Fatalf("root config must be confined before reading, got %d reads", readCount)
	}
}

func TestLoadConfigRootSymlinkAnchorsAtSpelledDir(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n")
	for _, dir := range []string{"env", "templates"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}
	writeTestFile(t, filepath.Join(root, "templates", "root.yaml"), strings.TrimSpace(`
imports:
  - ./common.yaml
services:
  local:
    type: $this.Service
`))
	writeTestFile(t, filepath.Join(root, "templates", "common.yaml"), "parameters: {which: templates}")
	writeTestFile(t, filepath.Join(root, "env", "common.yaml"), "parameters: {which: env}")
	rootPath := filepath.Join(root, "env", "gendi.yaml")
	if err := os.Symlink(filepath.Join("..", "templates", "root.yaml"), rootPath); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	cfg, err := loadConfigWithDefaultBoundary(t, rootPath)
	if err != nil {
		t.Fatalf("load root symlink: %v", err)
	}
	if got := cfg.Parameters["which"].Value.String(); got != "env" {
		t.Fatalf("relative import anchored at %q, want env context", got)
	}
	if got := cfg.Services["local"].Type; got != "example.com/app/env.Service" {
		t.Fatalf("$this resolved to %q, want addressed env package", got)
	}
}

// Cycle detection keys by real identity: a cycle reaching the root config
// through a different spelling (here, the root loaded via a symlink) is still
// detected.
func TestLoadConfigDetectsCycleThroughSymlinkedRoot(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "gendi.yaml"), strings.TrimSpace(`
imports:
  - ./child.yaml
`))
	writeTestFile(t, filepath.Join(dir, "child.yaml"), strings.TrimSpace(`
imports:
  - ./gendi.yaml
`))
	linkPath := filepath.Join(dir, "link.yaml")
	if err := os.Symlink("gendi.yaml", linkPath); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	_, err := loadConfigWithDefaultBoundary(t, linkPath)
	if err == nil {
		t.Fatal("expected cyclic import error")
	}
	if !strings.Contains(err.Error(), "cyclic import") || !strings.Contains(err.Error(), "gendi.yaml") {
		t.Fatalf("cycle must be detected at the root config, got: %v", err)
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

	cfg, err := loadConfigWithDefaultBoundary(t, rootPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if _, ok := cfg.Services["module.service"]; !ok {
		t.Fatalf("expected service from own-module import to load")
	}
}

// An import that escapes the module root via ".." is rejected.
func TestLoadConfigRejectsEscapingImport(t *testing.T) {
	for _, tt := range []struct {
		name       string
		importPath string
	}{
		{name: "parent path", importPath: "../secret.yaml"},
		{
			// The dotted first segment looks module-shaped, but the path is
			// still a local escape and must be rejected in the same way.
			name:       "dotted first segment",
			importPath: "assets.d/../../secret.yaml",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			outer := t.TempDir()
			writeTestFile(t, filepath.Join(outer, "secret.yaml"), "parameters: {secret: leaked}")

			moduleRoot := filepath.Join(outer, "module")
			if err := os.MkdirAll(moduleRoot, 0o755); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
			writeTestFile(t, filepath.Join(moduleRoot, "go.mod"), "module example.com/app\n")

			rootPath := filepath.Join(moduleRoot, "root.yaml")
			writeTestFile(t, rootPath, fmt.Sprintf("imports:\n  - path: %s\n", tt.importPath))

			if _, err := loadConfigWithDefaultBoundary(t, rootPath); err == nil {
				t.Fatal("expected error for import escaping the module root")
			}
		})
	}
}

func TestLoadConfigRejectsImportedSymlinkOutsideBoundary(t *testing.T) {
	outer := t.TempDir()
	external := filepath.Join(outer, "external.yaml")
	writeTestFile(t, external, "parameters: {secret: leaked}")

	moduleRoot := filepath.Join(outer, "module")
	if err := os.MkdirAll(moduleRoot, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeTestFile(t, filepath.Join(moduleRoot, "go.mod"), "module example.com/app\n")
	if err := os.Symlink(external, filepath.Join(moduleRoot, "link.yaml")); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	rootPath := filepath.Join(moduleRoot, "root.yaml")
	writeTestFile(t, rootPath, "imports: [./link.yaml]")

	if _, err := loadConfigWithDefaultBoundary(t, rootPath); err == nil || !strings.Contains(err.Error(), "outside boundary") {
		t.Fatalf("expected imported config confinement error, got %v", err)
	}
}

func TestLoadConfigOutsideModuleUsesSeparateModuleContext(t *testing.T) {
	projectRoot := t.TempDir()
	writeTestFile(t, filepath.Join(projectRoot, "go.mod"), "module example.com/app\n")
	writeTestFile(t, filepath.Join(projectRoot, "services.yaml"), "parameters: {source: module}")

	configRoot := t.TempDir()
	rootPath := filepath.Join(configRoot, "gendi.yaml")
	writeTestFile(t, rootPath, "imports: [example.com/app/services.yaml]")

	boundary, err := DefaultBoundary(rootPath)
	if err != nil {
		t.Fatalf("default boundary: %v", err)
	}
	cfg, err := LoadConfig(rootPath, boundary, projectRoot)
	if err != nil {
		t.Fatalf("load external config with module context: %v", err)
	}
	if got := cfg.Parameters["source"].Value.String(); got != "module" {
		t.Fatalf("source = %q, want module", got)
	}
}

func TestLoadConfigNestedModuleRequiresModulePath(t *testing.T) {
	root := t.TempDir()
	tools := filepath.Join(root, "tools")
	if err := os.MkdirAll(tools, 0o755); err != nil {
		t.Fatalf("mkdir tools: %v", err)
	}
	writeTestFile(t, filepath.Join(root, "go.mod"), strings.TrimSpace(`
module example.com/app

go 1.24

require example.com/tools v0.0.0

replace example.com/tools => ./tools
`))
	writeTestFile(t, filepath.Join(tools, "go.mod"), strings.TrimSpace(`
module example.com/tools

go 1.24
`))
	writeTestFile(t, filepath.Join(tools, "gendi.yaml"), "parameters: {tool: loaded}")

	localRoot := filepath.Join(root, "local.yaml")
	writeTestFile(t, localRoot, "imports: [./tools/gendi.yaml]")
	if _, err := loadConfigWithDefaultBoundary(t, localRoot); err == nil ||
		!strings.Contains(err.Error(), "crosses Go module boundary") ||
		!strings.Contains(err.Error(), "module-path import") {
		t.Fatalf("expected local nested-module import to fail, got %v", err)
	}

	moduleRoot := filepath.Join(root, "module.yaml")
	writeTestFile(t, moduleRoot, "imports: [example.com/tools/gendi.yaml]")
	cfg, err := loadConfigWithDefaultBoundary(t, moduleRoot)
	if err != nil {
		t.Fatalf("module-path import: %v", err)
	}
	if got := cfg.Parameters["tool"].Value.String(); got != "loaded" {
		t.Fatalf("tool = %q, want loaded", got)
	}
}

func TestLoadConfigExcludesImportedSymlinkBeforeConfinement(t *testing.T) {
	outer := t.TempDir()
	externalDir := filepath.Join(outer, "external")
	if err := os.MkdirAll(externalDir, 0o755); err != nil {
		t.Fatalf("mkdir external: %v", err)
	}
	writeTestFile(t, filepath.Join(externalDir, "secret.yaml"), "parameters: {secret: leaked}")

	moduleRoot := filepath.Join(outer, "module")
	servicesDir := filepath.Join(moduleRoot, "services")
	if err := os.MkdirAll(servicesDir, 0o755); err != nil {
		t.Fatalf("mkdir services: %v", err)
	}
	writeTestFile(t, filepath.Join(moduleRoot, "go.mod"), "module example.com/app\n")
	writeTestFile(t, filepath.Join(servicesDir, "app.yaml"), "parameters: {app: loaded}")
	if err := os.Symlink(externalDir, filepath.Join(servicesDir, "fixtures")); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	rootPath := filepath.Join(moduleRoot, "root.yaml")
	writeTestFile(t, rootPath, strings.TrimSpace(`
imports:
  - path: ./services/**/*.yaml
    exclude: [./services/fixtures]
`))

	cfg, err := loadConfigWithDefaultBoundary(t, rootPath)
	if err != nil {
		t.Fatalf("load with excluded symlink: %v", err)
	}
	if _, ok := cfg.Parameters["app"]; !ok {
		t.Fatal("expected regular candidate to load")
	}
	if _, ok := cfg.Parameters["secret"]; ok {
		t.Fatal("excluded external candidate must not load")
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

	cfg, err := loadConfigWithDefaultBoundary(t, rootPath)
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
	nestedDir := filepath.Join(dir, "configs", "nested")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatalf("create nested dir: %v", err)
	}

	writeTestFile(t, filepath.Join(nestedDir, "service.yaml"), strings.TrimSpace(`
services:
  nested:
    constructor:
      func: "example.NewNested"
`))

	rootPath := filepath.Join(dir, "root.yaml")
	writeTestFile(t, rootPath, strings.TrimSpace(`
imports:
  - "./configs/**/*.yaml"
`))

	cfg, err := loadConfigWithDefaultBoundary(t, rootPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if _, ok := cfg.Services["nested"]; !ok {
		t.Fatal("expected service from nested directory to load")
	}
}

func TestLoadConfigOwnModuleImportGlob(t *testing.T) {
	for _, tt := range []struct {
		name         string
		importPath   string
		wantServices []string
	}{
		{
			name:         "direct children",
			importPath:   "example.com/app/imports/*.yaml",
			wantServices: []string{"module.service", "module.extra"},
		},
		{
			name:         "recursive",
			importPath:   "example.com/app/imports/**/*.yaml",
			wantServices: []string{"module.service", "module.extra", "module.nested"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeModuleImportsFixture(t, dir)

			rootPath := filepath.Join(dir, "root.yaml")
			writeTestFile(t, rootPath, fmt.Sprintf("imports:\n  - path: %s\n", tt.importPath))

			cfg, err := loadConfigWithDefaultBoundary(t, rootPath)
			if err != nil {
				t.Fatalf("load config: %v", err)
			}
			for _, service := range tt.wantServices {
				if _, ok := cfg.Services[service]; !ok {
					t.Errorf("expected service %q to load", service)
				}
			}
		})
	}
}

func TestLoadConfigServiceAlias(t *testing.T) {
	configPath := filepath.Join(getCurrentDir(), "testdata", "service_alias", "gendi.yaml")
	cfg, err := loadConfigWithDefaultBoundary(t, configPath)
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

func TestLoadConfigNullArgument(t *testing.T) {
	configPath := filepath.Join(getCurrentDir(), "testdata", "null_argument", "gendi.yaml")
	cfg, err := loadConfigWithDefaultBoundary(t, configPath)
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
