package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Module info is derived from the output location — the generated file's
// package identity is defined by the module that will contain it — never
// from the process working directory.
func TestFinalizeDerivesModuleFromOutput(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeTestGoMod(t, root, "module example.com/app\n")

	opts := Options{Out: filepath.Join(root, "internal", "di"), Package: "di"}
	if err := opts.Finalize(); err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	if opts.ModulePath != "example.com/app" {
		t.Errorf("ModulePath = %q, want example.com/app", opts.ModulePath)
	}
	if opts.ModuleRoot != root {
		t.Errorf("ModuleRoot = %q, want %q", opts.ModuleRoot, root)
	}
	if opts.OutputPkgPath != "example.com/app/internal/di" {
		t.Errorf("OutputPkgPath = %q, want example.com/app/internal/di", opts.OutputPkgPath)
	}
}

// A go.mod whose module line cannot be parsed is skipped and the walk
// continues upward — the same rule the import sandbox uses
// (gomod.FindModuleRoot), so loading and codegen agree on the module.
func TestFinalizeSkipsUnparseableGoMod(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeTestGoMod(t, root, "module example.com/app\n")
	nested := filepath.Join(root, "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeTestGoMod(t, nested, "// no module line here\n")

	opts := Options{Out: filepath.Join(nested, "di"), Package: "di"}
	if err := opts.Finalize(); err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	if opts.ModulePath != "example.com/app" {
		t.Errorf("ModulePath = %q, want example.com/app", opts.ModulePath)
	}
	if opts.OutputPkgPath != "example.com/app/nested/di" {
		t.Errorf("OutputPkgPath = %q, want example.com/app/nested/di", opts.OutputPkgPath)
	}
}

// An output location outside any Go module cannot define a package identity.
func TestFinalizeRequiresModuleAboveOutput(t *testing.T) {
	t.Parallel()

	opts := Options{Out: filepath.Join(t.TempDir(), "di"), Package: "di"}
	err := opts.Finalize()
	if err == nil || !strings.Contains(err.Error(), "go.mod not found above output") {
		t.Fatalf("expected 'go.mod not found above output' error, got %v", err)
	}
}

func writeTestGoMod(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(content), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
}
