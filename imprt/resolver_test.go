package imprt

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestClassify(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		pattern string
		want    kind
		wantErr bool
	}{
		{name: "explicit relative", pattern: "./services/app.yaml", want: kindLocal},
		{name: "explicit parent", pattern: "../services/app.yaml", want: kindLocal},
		{name: "plain relative", pattern: "services/app.yaml", want: kindLocal},
		{name: "plain relative glob", pattern: "services/*.yaml", want: kindLocal},
		{name: "explicit relative dotted segment", pattern: "./assets.d/app.yaml", want: kindLocal},
		{name: "dotted first segment", pattern: "assets.d/app.yaml", want: kindModule},
		{name: "single-segment dotted file", pattern: "base.yaml", want: kindLocal},
		{name: "single-segment glob", pattern: "test_*.yaml", want: kindLocal},
		{name: "dot", pattern: ".", want: kindLocal},
		{name: "dot-dot", pattern: "..", want: kindLocal},
		{name: "module path", pattern: "example.com/mod/app.yaml", want: kindModule},
		{name: "module glob", pattern: "example.com/mod/*.yaml", want: kindModule},
		{name: "empty", pattern: "", wantErr: true},
		{name: "absolute", pattern: string(filepath.Separator) + "etc/app.yaml", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := classify(tt.pattern)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got kind %v", tt.pattern, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("classify(%q): %v", tt.pattern, err)
			}
			if got != tt.want {
				t.Fatalf("classify(%q) = %v, want %v", tt.pattern, got, tt.want)
			}
		})
	}
}

func TestNewResolverRejectsEmptyBoundary(t *testing.T) {
	t.Parallel()

	if _, err := NewResolver(""); err == nil {
		t.Fatal("expected error for empty boundary")
	}
}

func TestResolveImportErrors(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	resolver := newTestResolver(t, tempDir)

	_, err := resolver.ResolveImport(tempDir, "", nil)
	if err == nil || !strings.Contains(err.Error(), "import path is empty") {
		t.Fatalf("expected empty path error, got %v", err)
	}

	_, err = resolver.ResolveImport(tempDir, "./missing.yaml", nil)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestResolveImportLocalFile(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	resolver := newTestResolver(t, tempDir)

	path := filepath.Join(tempDir, "config.yaml")
	writeFile(t, path, "content")

	result, err := resolver.ResolveImport(tempDir, "./config.yaml", nil)
	if err != nil {
		t.Fatalf("resolve local failed: %v", err)
	}
	expected := []string{mustAbs(t, path)}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestResolveImportLocalGlob(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	resolver := newTestResolver(t, tempDir)

	writeFile(t, filepath.Join(tempDir, "a.yaml"), "a")
	writeFile(t, filepath.Join(tempDir, "b.yaml"), "b")

	result, err := resolver.ResolveImport(tempDir, "./*.yaml", nil)
	if err != nil {
		t.Fatalf("resolve glob failed: %v", err)
	}
	expected := []string{
		mustAbs(t, filepath.Join(tempDir, "a.yaml")),
		mustAbs(t, filepath.Join(tempDir, "b.yaml")),
	}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

// Glob metacharacters in the checkout path (the anchor directory) are literal
// path bytes, never pattern syntax.
func TestResolveImportGlobMetacharAnchor(t *testing.T) {
	t.Parallel()

	tempDir := filepath.Join(t.TempDir(), "work[2024]", "app")
	writeFile(t, filepath.Join(tempDir, "services", "a.yaml"), "a")
	writeFile(t, filepath.Join(tempDir, "services", "b.yaml"), "b")
	resolver := newTestResolver(t, tempDir)

	result, err := resolver.ResolveImport(tempDir, "./services/*.yaml", nil)
	if err != nil {
		t.Fatalf("resolve glob failed: %v", err)
	}
	expected := []string{
		mustAbs(t, filepath.Join(tempDir, "services", "a.yaml")),
		mustAbs(t, filepath.Join(tempDir, "services", "b.yaml")),
	}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

// A glob that matches nothing — including one whose base directory does not
// exist — is a silent no-op, not an error.
func TestResolveImportGlobNoMatchIsSilent(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	resolver := newTestResolver(t, tempDir)

	result, err := resolver.ResolveImport(tempDir, "./missing-dir/*.yaml", nil)
	if err != nil {
		t.Fatalf("no-match glob must be silent, got %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected no files, got %v", result)
	}
}

// A bare module import (no file) is rejected: an import must name its file
// explicitly rather than falling back to a guessed gendi.yaml.
func TestResolveImportRejectsBareModuleImport(t *testing.T) {
	t.Parallel()

	moduleRoot, baseDir, modulePath := createModule(t)
	writeFile(t, filepath.Join(moduleRoot, "gendi.yaml"), "name: module")
	resolver := newTestResolver(t, moduleRoot)

	if _, err := resolver.ResolveImport(baseDir, modulePath, nil); err == nil {
		t.Fatal("expected error for bare module import without an explicit file")
	}
}

func TestResolveImportModuleFile(t *testing.T) {
	t.Parallel()

	moduleRoot, baseDir, modulePath := createModule(t)
	configPath := filepath.Join(moduleRoot, "configs", "app.yaml")
	writeFile(t, configPath, "name: app")
	resolver := newTestResolver(t, moduleRoot)

	result, err := resolver.ResolveImport(baseDir, modulePath+"/configs/app.yaml", nil)
	if err != nil {
		t.Fatalf("resolve module file failed: %v", err)
	}
	expected := []string{mustAbs(t, configPath)}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestResolveImportModuleFileMissing(t *testing.T) {
	t.Parallel()

	moduleRoot, baseDir, modulePath := createModule(t)
	resolver := newTestResolver(t, moduleRoot)

	_, err := resolver.ResolveImport(baseDir, modulePath+"/configs/missing.yaml", nil)
	if err == nil || !strings.Contains(err.Error(), "does not contain") {
		t.Fatalf("expected missing module file error, got %v", err)
	}
}

func TestResolveImportModuleGlob(t *testing.T) {
	t.Parallel()

	moduleRoot, baseDir, modulePath := createModule(t)
	writeFile(t, filepath.Join(moduleRoot, "configs", "a.yaml"), "a")
	writeFile(t, filepath.Join(moduleRoot, "configs", "b.yaml"), "b")
	resolver := newTestResolver(t, moduleRoot)

	result, err := resolver.ResolveImport(baseDir, modulePath+"/configs/*.yaml", nil)
	if err != nil {
		t.Fatalf("resolve module glob failed: %v", err)
	}
	expected := []string{
		mustAbs(t, filepath.Join(moduleRoot, "configs", "a.yaml")),
		mustAbs(t, filepath.Join(moduleRoot, "configs", "b.yaml")),
	}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestResolveImportRejectsAbsolutePath(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	resolver := newTestResolver(t, tempDir)

	_, err := resolver.ResolveImport(tempDir, filepath.Join(t.TempDir(), "x.yaml"), nil)
	if err == nil || !strings.Contains(err.Error(), "absolute") {
		t.Fatalf("expected absolute-path rejection, got %v", err)
	}
}

// A multi-segment path whose first segment contains a dot is classified as a
// module path, deterministically — even when a matching local directory
// exists. The error hints at the ./ spelling for local directories.
func TestResolveImportDottedSegmentIsModule(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(root, "assets.d", "app.yaml"), "x: 1")
	resolver := newTestResolver(t, root)

	_, err := resolver.ResolveImport(root, "assets.d/app.yaml", nil)
	if err == nil || !strings.Contains(err.Error(), "./assets.d/app.yaml") {
		t.Fatalf("expected module-not-found error hinting the ./ spelling, got %v", err)
	}

	// The explicit relative spelling resolves the same layout.
	result, err := resolver.ResolveImport(root, "./assets.d/app.yaml", nil)
	if err != nil {
		t.Fatalf("explicit relative spelling failed: %v", err)
	}
	expected := []string{mustAbs(t, filepath.Join(root, "assets.d", "app.yaml"))}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

// A single-segment pattern can never be a valid module import (a module
// import must name a file inside the module), so a bare dotted filename like
// base.yaml resolves locally — the master spelling keeps working.
func TestResolveImportSingleSegmentDottedFile(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	resolver := newTestResolver(t, tempDir)
	path := filepath.Join(tempDir, "base.yaml")
	writeFile(t, path, "x: 1")

	result, err := resolver.ResolveImport(tempDir, "base.yaml", nil)
	if err != nil {
		t.Fatalf("bare dotted filename should resolve locally: %v", err)
	}
	if !reflect.DeepEqual(result, []string{mustAbs(t, path)}) {
		t.Fatalf("got %v, want %v", result, []string{mustAbs(t, path)})
	}
}

// A relative import escaping the module of the importing file is rejected.
func TestResolveImportRejectsRelativeEscape(t *testing.T) {
	t.Parallel()

	outer := t.TempDir()
	writeFile(t, filepath.Join(outer, "secret.yaml"), "secret: leaked")
	root := filepath.Join(outer, "module")
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n")
	resolver := newTestResolver(t, root)

	if _, err := resolver.ResolveImport(root, "../secret.yaml", nil); err == nil {
		t.Fatal("expected error for import escaping the module root")
	}
	if _, err := resolver.ResolveImport(root, "./../secret.yaml", nil); err == nil {
		t.Fatal("expected error for cleaned ../ chain escaping the module root")
	}
	if _, err := resolver.ResolveImport(root, "../*.yaml", nil); err == nil {
		t.Fatal("expected error for glob pattern escaping the module root")
	}
}

// A module-path import whose remainder uses "../" to climb out of the resolved
// module must be rejected, even though the path is genuinely module-shaped.
func TestResolveImportRejectsModuleRemainderEscape(t *testing.T) {
	t.Parallel()

	moduleRoot, baseDir, modulePath := createModule(t)
	writeFile(t, filepath.Join(filepath.Dir(moduleRoot), "secret.yaml"), "secret: leaked")
	resolver := newTestResolver(t, moduleRoot)

	if _, err := resolver.ResolveImport(baseDir, modulePath+"/../secret.yaml", nil); err == nil {
		t.Fatal("expected error for module remainder escaping the module")
	}
}

// A file resolves siblings within its OWN module, even when that module differs
// from the fallback boundary — proving the boundary is the importing file's
// module, so a dependency's config can reference its own siblings.
func TestResolveImportConfinesToImportingModuleNotFallback(t *testing.T) {
	t.Parallel()

	fallback := t.TempDir() // stands in for the main/root module
	dep := t.TempDir()      // a separate module tree
	writeFile(t, filepath.Join(dep, "go.mod"), "module example.com/dep\n")
	sibling := filepath.Join(dep, "sibling.yaml")
	writeFile(t, sibling, "x: 1")

	resolver := newTestResolver(t, fallback)
	got, err := resolver.ResolveImport(dep, "./sibling.yaml", nil)
	if err != nil {
		t.Fatalf("dep sibling should resolve within its own module: %v", err)
	}
	if !reflect.DeepEqual(got, []string{mustAbs(t, sibling)}) {
		t.Fatalf("got %v, want %v", got, []string{mustAbs(t, sibling)})
	}
}

// Exclusions are masks over the files the import found: a file mask drops
// matching files, and a mask matching a directory on a file's path — literally
// or via a glob — drops the whole subtree.
func TestResolveImportExcludeMasks(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	resolver := newTestResolver(t, tempDir)
	app := filepath.Join(tempDir, "services", "app.yaml")
	writeFile(t, app, "x: 1")
	writeFile(t, filepath.Join(tempDir, "services", "test_helper.yaml"), "x: 1")
	writeFile(t, filepath.Join(tempDir, "services", "internal", "skip.yaml"), "x: 1")
	writeFile(t, filepath.Join(tempDir, "services", "intro.yaml"), "x: 1")

	// File mask.
	got, err := resolver.ResolveImport(tempDir, "./services/**/*.yaml", []string{"./services/test_*.yaml"})
	if err != nil {
		t.Fatalf("resolve with file mask: %v", err)
	}
	want := []string{
		mustAbs(t, app),
		mustAbs(t, filepath.Join(tempDir, "services", "internal", "skip.yaml")),
		mustAbs(t, filepath.Join(tempDir, "services", "intro.yaml")),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("file mask: got %v, want %v", got, want)
	}

	// Literal directory mask excludes the subtree.
	got, err = resolver.ResolveImport(tempDir, "./services/**/*.yaml", []string{"./services/internal", "./services/test_*.yaml"})
	if err != nil {
		t.Fatalf("resolve with dir mask: %v", err)
	}
	want = []string{mustAbs(t, app), mustAbs(t, filepath.Join(tempDir, "services", "intro.yaml"))}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("dir mask: got %v, want %v", got, want)
	}

	// A glob mask matching a directory excludes its subtree, and matching
	// files still works: int* removes both internal/ and intro.yaml.
	got, err = resolver.ResolveImport(tempDir, "./services/**/*.yaml", []string{"./services/int*", "./services/test_*.yaml"})
	if err != nil {
		t.Fatalf("resolve with dir glob mask: %v", err)
	}
	want = []string{mustAbs(t, app)}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("dir glob mask: got %v, want %v", got, want)
	}

	// A literal import is subject to masks too.
	got, err = resolver.ResolveImport(tempDir, "./services/app.yaml", []string{"./services/app.yaml"})
	if err != nil {
		t.Fatalf("resolve literal with mask: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected literal import fully excluded, got %v", got)
	}
}

// An exclusion mask is a filter over what the import found: a mask that
// matches nothing — a missing file, a nonexistent directory, or a path
// outside the import entirely — is a silent no-op, never an error.
func TestResolveImportExcludeNoMatchIsNoOp(t *testing.T) {
	t.Parallel()

	outer := t.TempDir()
	root := filepath.Join(outer, "module")
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n")
	app := filepath.Join(root, "app.yaml")
	writeFile(t, app, "x: 1")
	resolver := newTestResolver(t, root)

	excludes := []string{"./missing.yaml", "./optional/dev_*.yaml", "../shared", "../.."}
	got, err := resolver.ResolveImport(root, "./*.yaml", excludes)
	if err != nil {
		t.Fatalf("non-matching masks must be no-ops: %v", err)
	}
	if !reflect.DeepEqual(got, []string{mustAbs(t, app)}) {
		t.Fatalf("got %v, want %v", got, []string{mustAbs(t, app)})
	}
}

// An exclusion must be addressed the same way as its import.
func TestResolveImportExcludeFormMustMatch(t *testing.T) {
	t.Parallel()

	moduleRoot, baseDir, modulePath := createModule(t)
	writeFile(t, filepath.Join(moduleRoot, "configs", "app.yaml"), "x: 1")
	resolver := newTestResolver(t, moduleRoot)

	_, err := resolver.ResolveImport(baseDir, modulePath+"/configs/*.yaml", []string{"./configs/app.yaml"})
	if err == nil || !strings.Contains(err.Error(), "does not match the addressing") {
		t.Fatalf("expected form-mismatch error for local exclude on module import, got %v", err)
	}

	_, err = resolver.ResolveImport(moduleRoot, "./configs/*.yaml", []string{modulePath + "/configs/app.yaml"})
	if err == nil || !strings.Contains(err.Error(), "does not match the addressing") {
		t.Fatalf("expected form-mismatch error for module exclude on local import, got %v", err)
	}
}

// A module import takes module-form masks addressing the SAME module; the
// mask filters by the path inside that module.
func TestResolveImportModuleExcludes(t *testing.T) {
	t.Parallel()

	moduleRoot, baseDir, modulePath := createModule(t)
	app := filepath.Join(moduleRoot, "configs", "app.yaml")
	writeFile(t, app, "x: 1")
	writeFile(t, filepath.Join(moduleRoot, "configs", "skip.yaml"), "x: 1")
	resolver := newTestResolver(t, moduleRoot)

	got, err := resolver.ResolveImport(baseDir, modulePath+"/configs/*.yaml", []string{modulePath + "/configs/skip.yaml"})
	if err != nil {
		t.Fatalf("module exclude failed: %v", err)
	}
	if !reflect.DeepEqual(got, []string{mustAbs(t, app)}) {
		t.Fatalf("got %v, want %v", got, []string{mustAbs(t, app)})
	}

	// A module-form mask naming a different module cannot match anything the
	// import found — reject it loudly instead of silently ignoring it.
	_, err = resolver.ResolveImport(baseDir, modulePath+"/configs/*.yaml", []string{"example.com/other/configs/skip.yaml"})
	if err == nil || !strings.Contains(err.Error(), "inside module") {
		t.Fatalf("expected error for exclude addressing another module, got %v", err)
	}
}

// A malformed exclusion mask is a loud error even when it would match nothing.
func TestResolveImportExcludeMalformedMask(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	resolver := newTestResolver(t, tempDir)
	writeFile(t, filepath.Join(tempDir, "app.yaml"), "x: 1")

	if _, err := resolver.ResolveImport(tempDir, "./*.yaml", []string{"[invalid"}); err == nil {
		t.Fatal("expected error for malformed exclusion mask")
	}
}

// A symlink whose real target is inside the module works like a regular file;
// any file whose real path lands outside its boundary is an error — for
// literals and globs alike. The escape hatch is explicit: exclude the
// escaping path, and it never reaches the sandbox check.
func TestResolveImportSymlinkEscape(t *testing.T) {
	t.Parallel()

	outer := t.TempDir()
	writeFile(t, filepath.Join(outer, "secrets", "creds.yaml"), "secret: leaked")
	root := filepath.Join(outer, "module")
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n")
	if err := os.Symlink(filepath.Join(outer, "secrets"), filepath.Join(root, "link")); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	resolver := newTestResolver(t, root)

	if _, err := resolver.ResolveImport(root, "./link/creds.yaml", nil); err == nil {
		t.Fatal("expected error for literal import through a symlink escaping the module")
	}
	if _, err := resolver.ResolveImport(root, "./link/*.yaml", nil); err == nil {
		t.Fatal("expected error for glob through a symlink escaping the module")
	}

	// Excluding the escaping path removes it before the sandbox check.
	got, err := resolver.ResolveImport(root, "./link/*.yaml", []string{"./link"})
	if err != nil {
		t.Fatalf("excluded symlink must not trip the sandbox: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no files, got %v", got)
	}
}

// A broad recursive glob sweeping an out-pointing symlinked directory is an
// error by default — and loads cleanly once the symlink is excluded.
func TestResolveImportGlobSymlinkEscapeExcludable(t *testing.T) {
	t.Parallel()

	outer := t.TempDir()
	writeFile(t, filepath.Join(outer, "shared", "extra.yaml"), "x: 1")
	root := filepath.Join(outer, "module")
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n")
	app := filepath.Join(root, "services", "app.yaml")
	writeFile(t, app, "x: 1")
	if err := os.Symlink(filepath.Join(outer, "shared"), filepath.Join(root, "services", "fixtures")); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	resolver := newTestResolver(t, root)

	if _, err := resolver.ResolveImport(root, "./services/**/*.yaml", nil); err == nil {
		t.Fatal("expected error for glob sweeping a symlink escaping the module")
	}

	got, err := resolver.ResolveImport(root, "./services/**/*.yaml", []string{"./services/fixtures"})
	if err != nil {
		t.Fatalf("excluding the symlinked dir must fix the import: %v", err)
	}
	if !reflect.DeepEqual(got, []string{mustAbs(t, app)}) {
		t.Fatalf("got %v, want only %v", got, app)
	}
}

// A module reached through a symlinked path keeps working: boundary and files
// are both resolved to real paths before comparison.
func TestResolveImportThroughSymlinkedRoot(t *testing.T) {
	t.Parallel()

	outer := t.TempDir()
	real := filepath.Join(outer, "real")
	writeFile(t, filepath.Join(real, "go.mod"), "module example.com/app\n")
	config := filepath.Join(real, "config.yaml")
	writeFile(t, config, "x: 1")
	linked := filepath.Join(outer, "linked")
	if err := os.Symlink(real, linked); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	resolver := newTestResolver(t, linked)

	got, err := resolver.ResolveImport(linked, "./config.yaml", nil)
	if err != nil {
		t.Fatalf("symlinked module root should keep working: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected one file, got %v", got)
	}
}

// A glob match that cannot be stat'ed — typically a dangling symlink left by
// an editor or deploy tool — is skipped, not a load-aborting error.
func TestResolveImportSkipsDanglingSymlink(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	resolver := newTestResolver(t, tempDir)
	good := filepath.Join(tempDir, "good.yaml")
	writeFile(t, good, "x: 1")
	if err := os.Symlink(filepath.Join(tempDir, "gone.yaml"), filepath.Join(tempDir, "link.yaml")); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	result, err := resolver.ResolveImport(tempDir, "./*.yaml", nil)
	if err != nil {
		t.Fatalf("dangling symlink must be skipped, got error: %v", err)
	}
	if !reflect.DeepEqual(result, []string{mustAbs(t, good)}) {
		t.Fatalf("got %v, want %v", result, []string{mustAbs(t, good)})
	}
}

// Module resolution is memoized per (baseDir, modulePath): resolving several
// imports of the same module must not re-run `go list` per import.
func TestResolverMemoizesModuleResolution(t *testing.T) {
	t.Parallel()

	moduleRoot, baseDir, modulePath := createModule(t)
	writeFile(t, filepath.Join(moduleRoot, "configs", "app.yaml"), "name: app")
	resolver := newTestResolver(t, moduleRoot)

	if _, err := resolver.ResolveImport(baseDir, modulePath+"/configs/app.yaml", nil); err != nil {
		t.Fatalf("resolve module file failed: %v", err)
	}
	if len(resolver.moduleDirs) == 0 {
		t.Fatal("expected module resolution to be memoized")
	}
	// The longest-first candidate walk tries example.com/testmod/configs/...
	// before finding the module: those failures must be memoized too, or
	// every import entry sharing the prefix re-runs the failing `go list`.
	negatives := 0
	for _, lookup := range resolver.moduleDirs {
		if !lookup.ok {
			negatives++
		}
	}
	if negatives == 0 {
		t.Fatal("expected failed candidate lookups to be memoized")
	}
	if _, err := resolver.ResolveImport(baseDir, modulePath+"/configs/*.yaml", nil); err != nil {
		t.Fatalf("second resolve failed: %v", err)
	}
}

func TestDefaultBoundary(t *testing.T) {
	t.Parallel()

	moduleRoot := t.TempDir()
	writeFile(t, filepath.Join(moduleRoot, "go.mod"), "module example.com/app\n")
	configPath := filepath.Join(moduleRoot, "app", "gendi.yaml")
	writeFile(t, configPath, "x: 1")

	got, err := DefaultBoundary(configPath)
	if err != nil {
		t.Fatalf("DefaultBoundary: %v", err)
	}
	if got != mustAbs(t, moduleRoot) {
		t.Fatalf("expected module root %s, got %s", moduleRoot, got)
	}

	outside := t.TempDir()
	outsideConfig := filepath.Join(outside, "gendi.yaml")
	writeFile(t, outsideConfig, "x: 1")

	got, err = DefaultBoundary(outsideConfig)
	if err != nil {
		t.Fatalf("DefaultBoundary: %v", err)
	}
	if got != mustAbs(t, outside) {
		t.Fatalf("expected config dir %s, got %s", outside, got)
	}
}

func newTestResolver(t *testing.T, boundary string) *Resolver {
	t.Helper()

	resolver, err := NewResolver(boundary)
	if err != nil {
		t.Fatalf("NewResolver(%q): %v", boundary, err)
	}
	return resolver
}

func createModule(t *testing.T) (moduleRoot string, baseDir string, modulePath string) {
	t.Helper()

	moduleRoot = t.TempDir()
	modulePath = "example.com/testmod"
	writeFile(t, filepath.Join(moduleRoot, "go.mod"), "module "+modulePath+"\n")

	baseDir = filepath.Join(moduleRoot, "subdir")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		t.Fatalf("failed to create base dir: %v", err)
	}

	return moduleRoot, baseDir, modulePath
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("failed to create dir %s: %v", dir, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
	}
}

func mustAbs(t *testing.T, path string) string {
	t.Helper()

	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("failed to abs path %s: %v", path, err)
	}
	return abs
}

// ResolveImport returns paths as addressed: an in-module symlink spelling is
// kept — it anchors the file's own relative imports and $this — while two
// spellings of the same real file still collapse into one entry.
func TestResolveImportKeepsSpelledPaths(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n")
	app := filepath.Join(root, "services", "real", "app.yaml")
	writeFile(t, app, "x: 1")
	if err := os.Symlink(filepath.Join(root, "services", "real"), filepath.Join(root, "services", "link")); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	resolver := newTestResolver(t, root)
	linkApp := filepath.Join(root, "services", "link", "app.yaml")

	got, err := resolver.ResolveImport(root, "./services/link/app.yaml", nil)
	if err != nil {
		t.Fatalf("resolve through in-module symlink: %v", err)
	}
	if !reflect.DeepEqual(got, []string{linkApp}) {
		t.Fatalf("got %v, want spelled path %v", got, []string{linkApp})
	}

	got, err = resolver.ResolveImport(root, "./services/**/*.yaml", nil)
	if err != nil {
		t.Fatalf("resolve glob: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("aliased spellings must collapse to one entry: got %v", got)
	}
}

// A module-shaped import whose spelling also exists relative to the importing
// file is ambiguous: loading either side would silently shadow the other, so
// it is a loud error. The ./ spelling stays the unambiguous way to the local
// path.
func TestResolveImportModuleLocalCollision(t *testing.T) {
	t.Parallel()

	moduleRoot, baseDir, modulePath := createModule(t)
	writeFile(t, filepath.Join(moduleRoot, "configs", "app.yaml"), "from: module")
	localMirror := filepath.Join(baseDir, modulePath, "configs", "app.yaml")
	writeFile(t, localMirror, "from: local")
	resolver := newTestResolver(t, moduleRoot)

	_, err := resolver.ResolveImport(baseDir, modulePath+"/configs/app.yaml", nil)
	if err == nil {
		t.Fatal("expected ambiguity error for module import shadowed by a local path")
	}
	if !strings.Contains(err.Error(), "ambiguous") || !strings.Contains(err.Error(), "./"+modulePath) {
		t.Fatalf("error must explain the ambiguity and the ./ spelling, got: %v", err)
	}

	_, err = resolver.ResolveImport(baseDir, modulePath+"/configs/*.yaml", nil)
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("expected ambiguity error for glob collision, got: %v", err)
	}

	got, err := resolver.ResolveImport(baseDir, "./"+modulePath+"/configs/app.yaml", nil)
	if err != nil {
		t.Fatalf("./ spelling must load the local file: %v", err)
	}
	if !reflect.DeepEqual(got, []string{mustAbs(t, localMirror)}) {
		t.Fatalf("got %v, want %v", got, []string{localMirror})
	}
}

// A relative baseDir confines exactly like its absolute spelling: the
// boundary is the module root of the importing directory, not the resolver's
// wider fallback. No t.Parallel — t.Chdir forbids it.
func TestResolveImportRelativeBaseDir(t *testing.T) {
	outer := t.TempDir()
	writeFile(t, filepath.Join(outer, "secret.yaml"), "secret: leaked")
	moduleRoot := filepath.Join(outer, "module")
	writeFile(t, filepath.Join(moduleRoot, "go.mod"), "module example.com/app\n")
	sub := filepath.Join(moduleRoot, "sub")
	app := filepath.Join(sub, "app.yaml")
	writeFile(t, app, "x: 1")
	resolver := newTestResolver(t, outer)

	t.Chdir(sub)

	got, err := resolver.ResolveImport(".", "./app.yaml", nil)
	if err != nil {
		t.Fatalf("relative baseDir must resolve local files: %v", err)
	}
	if !reflect.DeepEqual(got, []string{mustAbs(t, app)}) {
		t.Fatalf("got %v, want %v", got, []string{app})
	}

	if _, err := resolver.ResolveImport(".", "../../secret.yaml", nil); err == nil {
		t.Fatal("relative baseDir must not widen the confinement boundary past the module root")
	}
}

// Module resolution is a pure function of the importing file's module context
// and its go.mod graph — never of the process working directory. A module
// absent from the graph must not resolve just because the process happens to
// run inside a checkout of it.
func TestResolveImportModuleIgnoresWorkingDirectory(t *testing.T) {
	moduleRoot, baseDir, _ := createModule(t)
	resolver := newTestResolver(t, moduleRoot)

	otherCheckout := t.TempDir()
	writeFile(t, filepath.Join(otherCheckout, "go.mod"), "module example.com/other\n")
	writeFile(t, filepath.Join(otherCheckout, "x.yaml"), "secret: leaked")
	t.Chdir(otherCheckout)

	if _, err := resolver.ResolveImport(baseDir, "example.com/other/x.yaml", nil); err == nil {
		t.Fatal("expected error: example.com/other is not in the importing module's go.mod graph")
	}
}

// When the importing config lives outside any Go module, the explicitly
// supplied boundary is the module context: pointing it at a module root makes
// that module's imports resolve; anything less is a loud error asking for a
// project root.
func TestResolveImportModuleContextFromBoundary(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	writeFile(t, filepath.Join(projectRoot, "go.mod"), "module example.com/app\n")
	config := filepath.Join(projectRoot, "configs", "app.yaml")
	writeFile(t, config, "x: 1")

	outside := t.TempDir() // config dir outside any module
	resolver := newTestResolver(t, projectRoot)

	got, err := resolver.ResolveImport(outside, "example.com/app/configs/app.yaml", nil)
	if err != nil {
		t.Fatalf("boundary at a module root must provide the module context: %v", err)
	}
	if !reflect.DeepEqual(got, []string{mustAbs(t, config)}) {
		t.Fatalf("got %v, want %v", got, []string{mustAbs(t, config)})
	}

	// Neither baseDir nor boundary is inside a module → explicit error.
	noModule := newTestResolver(t, t.TempDir())
	_, err = noModule.ResolveImport(outside, "example.com/app/configs/app.yaml", nil)
	if err == nil || !strings.Contains(err.Error(), "requires a Go module") {
		t.Fatalf("expected 'requires a Go module' error, got %v", err)
	}
}
