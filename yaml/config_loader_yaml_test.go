package yaml

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gendi-org/gendi/imprt"
	"github.com/gendi-org/gendi/srcloc"
)

// boundaryFor derives the load boundary a LoadConfig caller must supply for a
// root config, exactly as production callers do.
func boundaryFor(t *testing.T, path string) string {
	t.Helper()
	boundary, err := DefaultBoundary(path)
	if err != nil {
		t.Fatalf("boundary for %s: %v", path, err)
	}
	return boundary
}

// defaultLoader builds a loader with the production resolver, confined to the
// module (or directory) of rootPath.
func defaultLoader(t *testing.T, rootPath string) *ConfigLoaderYaml {
	t.Helper()
	boundary := boundaryFor(t, rootPath)
	resolver, err := imprt.NewResolver(boundary, boundary)
	if err != nil {
		t.Fatalf("new resolver: %v", err)
	}
	return NewConfigLoaderYaml(resolver, NewParser())
}

type stubResolver struct {
	paths map[string][]string
}

func (r stubResolver) ResolveImport(baseDir, importPath string, _ []string) ([]imprt.Candidate, error) {
	if paths, ok := r.paths[importPath]; ok {
		candidates := make([]imprt.Candidate, 0, len(paths))
		for _, path := range paths {
			candidates = append(candidates, imprt.Candidate{
				Path:     path,
				Boundary: baseDir,
			})
		}
		return candidates, nil
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

func TestLoadDeduplicatesDiamondImports(t *testing.T) {
	dir := t.TempDir()

	rootPath := writeFile(t, dir, "root.yaml", `
imports:
  - path: b
  - path: c
parameters:
  a: "A"
`)
	bPath := writeFile(t, dir, "b.yaml", `
imports:
  - path: d
parameters:
  b: "B"
`)
	cPath := writeFile(t, dir, "c.yaml", `
imports:
  - path: d
parameters:
  c: "C"
`)
	dPath := writeFile(t, dir, "d.yaml", `
parameters:
  d: "D"
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

	cfg, err := loader.Load(rootPath, boundaryFor(t, rootPath))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got, want := len(cfg.Parameters), 4; got != want {
		t.Fatalf("expected %d parameters, got %d", want, got)
	}
	// d is reached through both b and c but resolves to one addressed path, so
	// it is loaded once: root, b, c, d — not d twice. A shared base reached
	// through many paths therefore costs one load, not one per path.
	if readCount != 4 {
		t.Fatalf("expected the shared import to be loaded once (4 reads), got %d", readCount)
	}
}

// Deduplication is keyed by the addressed path, so two symlink aliases of the
// same real file are still loaded independently — each keeps its own directory
// context for relative imports and $this.
func TestLoadDoesNotDeduplicateSymlinkAliases(t *testing.T) {
	dir := t.TempDir()

	realPath := writeFile(t, dir, "d.yaml", `
parameters:
  d: "D"
`)
	aliasPath := filepath.Join(dir, "d_alias.yaml")
	if err := os.Symlink(realPath, aliasPath); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}

	rootPath := writeFile(t, dir, "root.yaml", `
imports:
  - path: d
  - path: d_alias
`)

	loader := NewConfigLoaderYaml(stubResolver{
		paths: map[string][]string{
			"d":       {realPath},
			"d_alias": {aliasPath},
		},
	}, NewParser())

	readCount := 0
	origRead := defaultOsReadFile
	defaultOsReadFile = func(path string) ([]byte, error) {
		readCount++
		return os.ReadFile(path)
	}
	defer func() { defaultOsReadFile = origRead }()

	if _, err := loader.Load(rootPath, boundaryFor(t, rootPath)); err != nil {
		t.Fatalf("Load: %v", err)
	}
	// root, d (real spelling), d_alias (symlink spelling): the two spellings of
	// the same real file are distinct addressed paths and both load.
	if readCount != 3 {
		t.Fatalf("expected distinct symlink spellings to load independently (3 reads), got %d", readCount)
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
  app: "app"
`)
	writeFile(t, servicesDir, "db.yaml", `
parameters:
  db: "db"
`)
	writeFile(t, servicesDir, "test_helper.yaml", `
parameters:
  test_helper: "test"
`)

	rootPath := writeFile(t, dir, "gendi.yaml", `
imports:
  - path: ./services/*.yaml
    exclude:
      - ./services/test_*.yaml
`)

	loader := defaultLoader(t, rootPath)
	cfg, err := loader.Load(rootPath, boundaryFor(t, rootPath))
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
  app: "app"
`)
	writeFile(t, servicesDir, "test_app.yaml", `
parameters:
  test_app: "test"
`)
	writeFile(t, internalDir, "debug.yaml", `
parameters:
  debug: "debug"
`)

	rootPath := writeFile(t, dir, "gendi.yaml", `
imports:
  - path: ./services/**/*.yaml
    exclude:
      - ./services/test_*.yaml
      - ./services/internal/*.yaml
`)

	loader := defaultLoader(t, rootPath)
	cfg, err := loader.Load(rootPath, boundaryFor(t, rootPath))
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
  test1: "test1"
`)
	writeFile(t, servicesDir, "test2.yaml", `
parameters:
  test2: "test2"
`)

	rootPath := writeFile(t, dir, "gendi.yaml", `
imports:
  - path: ./services/*.yaml
    exclude:
      - ./services/*.yaml
`)

	loader := defaultLoader(t, rootPath)
	cfg, err := loader.Load(rootPath, boundaryFor(t, rootPath))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(cfg.Parameters) != 0 {
		t.Errorf("expected no parameters, got %d", len(cfg.Parameters))
	}
}

func TestExcludeNonGlobImport(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "specific.yaml", `
parameters:
  specific: "specific"
`)

	rootPath := writeFile(t, dir, "gendi.yaml", `
imports:
  - path: ./specific.yaml
    exclude:
      - ./specific.yaml
`)

	loader := defaultLoader(t, rootPath)

	cfg, err := loader.Load(rootPath, boundaryFor(t, rootPath))
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
  app: "app"
`)

	rootPath := writeFile(t, dir, "gendi.yaml", `
imports:
  - path: ./services/*.yaml
    exclude:
      - "[invalid"
`)

	loader := defaultLoader(t, rootPath)
	_, err := loader.Load(rootPath, boundaryFor(t, rootPath))
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
  app: "app"
`)
	writeFile(t, servicesDir, "db.yaml", `
parameters:
  db: "db"
`)

	// Test both scalar form and mapping form without exclude
	rootPath := writeFile(t, dir, "gendi.yaml", `
imports:
  - ./services/app.yaml
  - path: ./services/db.yaml
`)

	loader := defaultLoader(t, rootPath)
	cfg, err := loader.Load(rootPath, boundaryFor(t, rootPath))
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
  prod_db: "prod"
`)
	writeFile(t, devDir, "dev_database.yaml", `
parameters:
  dev_db: "dev"
`)
	writeFile(t, configDir, "base.yaml", `
parameters:
  base: "base"
`)

	rootPath := writeFile(t, dir, "gendi.yaml", `
imports:
  - path: ./config/**/*.yaml
    exclude:
      - ./config/**/dev_*.yaml
`)

	loader := defaultLoader(t, rootPath)
	cfg, err := loader.Load(rootPath, boundaryFor(t, rootPath))
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

// Exclusions are addressed like import paths, so absolute filesystem paths
// are rejected in them too.
func TestExcludeRejectsAbsolutePattern(t *testing.T) {
	dir := t.TempDir()
	servicesDir := filepath.Join(dir, "services")
	if err := os.MkdirAll(servicesDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	writeFile(t, servicesDir, "app.yaml", `
parameters:
  app: "app"
`)
	testPath := writeFile(t, servicesDir, "test.yaml", `
parameters:
  test: "test"
`)

	rootPath := writeFile(t, dir, "gendi.yaml", fmt.Sprintf(`
imports:
  - path: ./services/*.yaml
    exclude:
      - %q
`, testPath))

	loader := defaultLoader(t, rootPath)
	_, err := loader.Load(rootPath, boundaryFor(t, rootPath))
	if err == nil {
		t.Fatal("expected error for absolute exclusion pattern")
	}
	if !strings.Contains(err.Error(), "absolute") {
		t.Fatalf("error should mention absolute paths, got: %v", err)
	}
}

// A module import is excluded with a module-form pattern — the exclusion
// mirrors the import path's addressing, so no importer-relative or resolved-
// base guessing is involved.
func TestExcludeModuleImport(t *testing.T) {
	moduleRoot := t.TempDir()
	writeFile(t, moduleRoot, "go.mod", "module example.com/testmod\n")

	servicesDir := filepath.Join(moduleRoot, "services")
	if err := os.MkdirAll(servicesDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeFile(t, servicesDir, "app.yaml", `
parameters:
  app: "app"
`)
	writeFile(t, servicesDir, "skip.yaml", `
parameters:
  skip_me: "skip"
`)

	// The importing file lives in a subdirectory of the module, so its
	// directory differs from the module root the resolved files live under —
	// mirroring makes the exclusion independent of that difference.
	appDir := filepath.Join(moduleRoot, "app")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	rootPath := writeFile(t, appDir, "gendi.yaml", `
imports:
  - path: example.com/testmod/services/*.yaml
    exclude:
      - example.com/testmod/services/skip.yaml
`)

	loader := defaultLoader(t, rootPath)
	cfg, err := loader.Load(rootPath, boundaryFor(t, rootPath))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if _, ok := cfg.Parameters["app"]; !ok {
		t.Error("expected app parameter to be loaded")
	}
	if _, ok := cfg.Parameters["skip_me"]; ok {
		t.Error("expected skip_me parameter to be excluded")
	}
}

// An absolute pattern pointing at an existing directory is rejected like any
// other absolute exclusion — it does not silently exclude the subtree.
func TestExcludeRejectsAbsoluteDirectory(t *testing.T) {
	dir := t.TempDir()
	internalDir := filepath.Join(dir, "services", "internal")
	if err := os.MkdirAll(internalDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	writeFile(t, filepath.Join(dir, "services"), "app.yaml", `
parameters:
  app: "app"
`)
	writeFile(t, internalDir, "skip.yaml", `
parameters:
  skip_me: "skip"
`)

	rootPath := writeFile(t, dir, "gendi.yaml", fmt.Sprintf(`
imports:
  - path: ./services/**/*.yaml
    exclude:
      - %q
`, internalDir))

	loader := defaultLoader(t, rootPath)
	_, err := loader.Load(rootPath, boundaryFor(t, rootPath))
	if err == nil {
		t.Fatal("expected error for absolute directory exclusion")
	}
	if !strings.Contains(err.Error(), "absolute") {
		t.Fatalf("error should mention absolute paths, got: %v", err)
	}
}

// Absolute glob imports are rejected like any other absolute import path.
func TestLoadRejectsAbsoluteGlobImport(t *testing.T) {
	rootDir := t.TempDir()
	extDir := filepath.Join(rootDir, "ext")
	if err := os.MkdirAll(extDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeFile(t, extDir, "keep.yaml", `
parameters:
  keep: "keep"
`)

	rootPath := writeFile(t, rootDir, "gendi.yaml", fmt.Sprintf(`
imports:
  - path: %q
`, filepath.Join(extDir, "*.yaml")))

	loader := defaultLoader(t, rootPath)
	_, err := loader.Load(rootPath, boundaryFor(t, rootPath))
	if err == nil {
		t.Fatal("expected error for absolute glob import")
	}
	if !strings.Contains(err.Error(), "absolute") {
		t.Fatalf("error should mention absolute paths, got: %v", err)
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
		{
			name:     "service null body",
			yaml:     "services:\n  my_svc:",
			wantLine: 2,
			wantMsg:  "service must be a mapping or alias",
		},
		{
			name:     "default service null body",
			yaml:     "services:\n  _default:\n  my_svc:\n    constructor:\n      func: fmt.Println",
			wantLine: 2,
			wantMsg:  "service must be a mapping or alias",
		},
		{
			name:     "null tag entry",
			yaml:     "services:\n  my_svc:\n    constructor:\n      func: fmt.Println\n    tags:\n      -",
			wantLine: 6,
			wantMsg:  "tag name is required",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := writeFile(t, dir, "gendi.yaml", tt.yaml)

			loader := NewConfigLoaderYaml(stubResolver{}, NewParser())
			_, err := loader.Load(path, boundaryFor(t, path))
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
  app: "app"
`)
	writeFile(t, internalDir, "secret.yaml", `
parameters:
  secret: "secret"
`)
	writeFile(t, internalDir, "debug.yaml", `
parameters:
  debug: "debug"
`)

	rootPath := writeFile(t, dir, "gendi.yaml", `
imports:
  - path: ./services/**/*.yaml
    exclude:
      - ./services/internal
`)

	loader := defaultLoader(t, rootPath)
	cfg, err := loader.Load(rootPath, boundaryFor(t, rootPath))
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

// A glob exclusion that matches a directory excludes that directory's whole
// subtree (master behavior, regressed by the resolver-chain rerouting).
func TestExcludeGlobMatchingDirectory(t *testing.T) {
	dir := t.TempDir()
	internalDir := filepath.Join(dir, "services", "internal")
	if err := os.MkdirAll(internalDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	writeFile(t, filepath.Join(dir, "services"), "app.yaml", `
parameters:
  app: "app"
`)
	writeFile(t, internalDir, "skip.yaml", `
parameters:
  skip_me: "skip"
`)

	rootPath := writeFile(t, dir, "gendi.yaml", `
imports:
  - path: ./services/**/*.yaml
    exclude:
      - ./services/int*
`)

	loader := defaultLoader(t, rootPath)
	cfg, err := loader.Load(rootPath, boundaryFor(t, rootPath))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, ok := cfg.Parameters["app"]; !ok {
		t.Error("expected app parameter to be loaded")
	}
	if _, ok := cfg.Parameters["skip_me"]; ok {
		t.Error("expected files under the glob-matched directory to be excluded")
	}
}

// An exclusion is a mask over what the import found: a mask pointing outside
// the import (an existing sibling directory, an ancestor) matches none of the
// found files and is a silent no-op — never an error, never a way to blank
// the import.
func TestExcludeOutsideMaskIsNoOp(t *testing.T) {
	outer := t.TempDir()
	shared := filepath.Join(outer, "shared")
	if err := os.MkdirAll(shared, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	moduleRoot := filepath.Join(outer, "module")
	servicesDir := filepath.Join(moduleRoot, "services")
	if err := os.MkdirAll(servicesDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeFile(t, moduleRoot, "go.mod", "module example.com/app\n")
	writeFile(t, servicesDir, "app.yaml", `
parameters:
  app: "app"
`)

	rootPath := writeFile(t, moduleRoot, "gendi.yaml", `
imports:
  - path: ./services/*.yaml
    exclude:
      - ../shared
      - ../..
      - ./services/missing.yaml
`)

	loader := defaultLoader(t, rootPath)
	cfg, err := loader.Load(rootPath, boundaryFor(t, rootPath))
	if err != nil {
		t.Fatalf("non-matching masks must be no-ops: %v", err)
	}
	if _, ok := cfg.Parameters["app"]; !ok {
		t.Error("expected app parameter to be loaded")
	}
}

// An exclusion glob whose base directory does not exist is a silent no-op,
// like any other glob that matches nothing.
func TestExcludeGlobMissingBaseDirIsSilent(t *testing.T) {
	dir := t.TempDir()
	servicesDir := filepath.Join(dir, "services")
	if err := os.MkdirAll(servicesDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeFile(t, servicesDir, "app.yaml", `
parameters:
  app: "app"
`)

	rootPath := writeFile(t, dir, "gendi.yaml", `
imports:
  - path: ./services/*.yaml
    exclude:
      - ./optional/dev_*.yaml
`)

	loader := defaultLoader(t, rootPath)
	cfg, err := loader.Load(rootPath, boundaryFor(t, rootPath))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, ok := cfg.Parameters["app"]; !ok {
		t.Error("expected app parameter to be loaded")
	}
}

// An exclusion must be addressed the same way as its import: a local exclude
// on a module import (or a module exclude on a local import) is an error.
func TestExcludeFormMustMatchImportForm(t *testing.T) {
	moduleRoot := t.TempDir()
	writeFile(t, moduleRoot, "go.mod", "module example.com/testmod\n")
	servicesDir := filepath.Join(moduleRoot, "services")
	if err := os.MkdirAll(servicesDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeFile(t, servicesDir, "app.yaml", `
parameters:
  app: "app"
`)
	writeFile(t, servicesDir, "skip.yaml", `
parameters:
  skip_me: "skip"
`)

	// Module import + local exclude.
	rootPath := writeFile(t, moduleRoot, "gendi.yaml", `
imports:
  - path: example.com/testmod/services/*.yaml
    exclude:
      - ./services/skip.yaml
`)
	loader := defaultLoader(t, rootPath)
	if _, err := loader.Load(rootPath, boundaryFor(t, rootPath)); err == nil || !strings.Contains(err.Error(), "does not match the addressing") {
		t.Fatalf("expected form-mismatch error for local exclude on module import, got %v", err)
	}

	// Local import + module exclude.
	rootPath = writeFile(t, moduleRoot, "gendi.yaml", `
imports:
  - path: ./services/*.yaml
    exclude:
      - example.com/testmod/services/skip.yaml
`)
	loader = defaultLoader(t, rootPath)
	if _, err := loader.Load(rootPath, boundaryFor(t, rootPath)); err == nil || !strings.Contains(err.Error(), "does not match the addressing") {
		t.Fatalf("expected form-mismatch error for module exclude on local import, got %v", err)
	}
}

// An empty boundary is rejected loudly instead of silently degrading to the
// process working directory.
func TestLoadConfigRejectsEmptyBoundary(t *testing.T) {
	dir := t.TempDir()
	rootPath := writeFile(t, dir, "gendi.yaml", `
parameters:
  app: "app"
`)

	if _, err := LoadConfig(rootPath, "", ""); err == nil || !strings.Contains(err.Error(), "boundary") {
		t.Fatalf("expected empty-boundary error, got %v", err)
	}
}
