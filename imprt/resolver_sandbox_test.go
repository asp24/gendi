package imprt

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// stubInner records the arguments it was called with and returns canned
// results, so tests can observe how ResolverSandbox rewrites paths and can
// feed it files inside or outside the module root.
type stubInner struct {
	gotBaseDir string
	gotPath    string
	files      []string
	err        error
}

func (s *stubInner) CanResolve(string) bool { return true }

func (s *stubInner) Resolve(baseDir, importPath string) ([]string, error) {
	s.gotBaseDir = baseDir
	s.gotPath = importPath
	if s.err != nil {
		return nil, s.err
	}
	return s.files, nil
}

func writeGoMod(t *testing.T, dir, modulePath string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module "+modulePath+"\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
}

// An absolute import path is rejected outright — even one pointing inside the
// module. Everything in the module is addressable relatively, other modules
// are addressed by module path.
func TestSandboxRejectsAbsolutePath(t *testing.T) {
	root := t.TempDir()
	writeGoMod(t, root, "example.com/app")

	inside := filepath.Join(root, "services", "x.yaml")
	inner := &stubInner{files: []string{inside}}
	sandbox := NewResolverSandbox(inner, root)

	_, err := sandbox.Resolve(root, inside)
	if err == nil {
		t.Fatal("expected error for absolute import path")
	}
	if !strings.Contains(err.Error(), "absolute") {
		t.Errorf("error should mention absolute paths, got: %v", err)
	}
	if inner.gotPath != "" {
		t.Errorf("inner resolver was called with %q, want no call", inner.gotPath)
	}
}

func TestSandboxRejectsFileOutsideModuleRoot(t *testing.T) {
	root := t.TempDir()
	writeGoMod(t, root, "example.com/app")

	// Inner claims to resolve to a file outside the module root.
	inner := &stubInner{files: []string{filepath.Join(filepath.Dir(root), "escape.yaml")}}
	sandbox := NewResolverSandbox(inner, root)

	_, err := sandbox.Resolve(root, "./escape.yaml")
	if err == nil {
		t.Fatal("expected error for file resolved outside the module root")
	}
}

func TestSandboxAllowsRelativeWithinModuleRoot(t *testing.T) {
	root := t.TempDir()
	writeGoMod(t, root, "example.com/app")
	baseDir := filepath.Join(root, "app")

	want := filepath.Join(baseDir, "x.yaml")
	inner := &stubInner{files: []string{want}}
	sandbox := NewResolverSandbox(inner, root)

	files, err := sandbox.Resolve(baseDir, "./x.yaml")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !reflect.DeepEqual(files, []string{want}) {
		t.Errorf("files = %v, want %v", files, []string{want})
	}
	// Relative paths are passed through untouched.
	if inner.gotPath != "./x.yaml" {
		t.Errorf("inner got path %q, want %q", inner.gotPath, "./x.yaml")
	}
}

func TestSandboxExemptsModulePathImports(t *testing.T) {
	root := t.TempDir()
	writeGoMod(t, root, "example.com/app")

	// A module-path import may resolve into a dependency module outside the
	// main module root; the sandbox must not reject it or rewrite it.
	depFile := filepath.Join(filepath.Dir(root), "dep", "cfg.yaml")
	inner := &stubInner{files: []string{depFile}}
	sandbox := NewResolverSandbox(inner, root)

	files, err := sandbox.Resolve(root, "example.com/dep/cfg.yaml")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if inner.gotPath != "example.com/dep/cfg.yaml" {
		t.Errorf("module import was rewritten to %q", inner.gotPath)
	}
	if !reflect.DeepEqual(files, []string{depFile}) {
		t.Errorf("files = %v, want %v", files, []string{depFile})
	}
}

func TestSandboxUsesFallbackRootWithoutModule(t *testing.T) {
	// No go.mod anywhere up the tree from baseDir → the fallback root is the
	// containment boundary.
	root := t.TempDir()
	baseDir := filepath.Join(root, "sub")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	within := filepath.Join(root, "x.yaml")
	inner := &stubInner{files: []string{within}}
	sandbox := NewResolverSandbox(inner, root)

	files, err := sandbox.Resolve(baseDir, "../x.yaml")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !reflect.DeepEqual(files, []string{within}) {
		t.Errorf("files = %v, want %v", files, []string{within})
	}

	// A file resolving outside the fallback root is rejected.
	inner.files = []string{filepath.Join(filepath.Dir(root), "escape.yaml")}
	if _, err := sandbox.Resolve(baseDir, "../../escape.yaml"); err == nil {
		t.Fatal("expected error for file outside the fallback root")
	}
}

func TestSandboxPropagatesInnerError(t *testing.T) {
	root := t.TempDir()
	writeGoMod(t, root, "example.com/app")

	sentinel := errors.New("boom")
	inner := &stubInner{err: sentinel}
	sandbox := NewResolverSandbox(inner, root)

	if _, err := sandbox.Resolve(root, "./x.yaml"); !errors.Is(err, sentinel) {
		t.Fatalf("expected inner error to propagate, got %v", err)
	}
}
