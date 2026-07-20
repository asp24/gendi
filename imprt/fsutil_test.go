package imprt

import (
	"path/filepath"
	"testing"
)

// A glob that matches nothing is a silent no-op — whether its base directory
// exists or not. Only a malformed pattern is an error.
func TestGlobMatches(t *testing.T) {
	dir := t.TempDir()

	files, err := globMatches(dir, "*.yaml")
	if err != nil {
		t.Fatalf("unexpected error for empty match: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("expected no matches, got %v", files)
	}

	files, err = globMatches(dir, "no_such_dir/*.yaml")
	if err != nil {
		t.Fatalf("unexpected error for missing base directory: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("expected no matches, got %v", files)
	}

	if _, err = globMatches(dir, "[invalid"); err == nil {
		t.Fatal("expected error for malformed pattern")
	}

	writeFile(t, filepath.Join(dir, "sub", "a.yaml"), "a")
	writeFile(t, filepath.Join(dir, "b.yaml"), "b")
	files, err = globMatches(dir, "*")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{mustAbs(t, filepath.Join(dir, "b.yaml"))}
	if len(files) != 1 || files[0] != want[0] {
		t.Fatalf("expected files %v, got %v", want, files)
	}
}

// Glob metacharacters in the anchor directory are literal path bytes, not
// pattern syntax.
func TestGlobMatchesMetacharInRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "work[2024]", "app")
	writeFile(t, filepath.Join(root, "services", "a.yaml"), "a")
	writeFile(t, filepath.Join(root, "services", "b.yaml"), "b")

	files, err := globMatches(root, "services/*.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{
		mustAbs(t, filepath.Join(root, "services", "a.yaml")),
		mustAbs(t, filepath.Join(root, "services", "b.yaml")),
	}
	if len(files) != 2 || files[0] != want[0] || files[1] != want[1] {
		t.Fatalf("expected files %v, got %v", want, files)
	}
}

// Native path separators in a glob are normalized before doublestar parses
// the pattern. In particular, Windows separators must not be read as escapes.
func TestGlobMatchesNativeSeparators(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "services", "a.yaml"), "a")
	writeFile(t, filepath.Join(root, "services", "nested", "b.yaml"), "b")

	files, err := globMatches(root, filepath.Join("services", "**", "*.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{
		mustAbs(t, filepath.Join(root, "services", "a.yaml")),
		mustAbs(t, filepath.Join(root, "services", "nested", "b.yaml")),
	}
	if len(files) != 2 || files[0] != want[0] || files[1] != want[1] {
		t.Fatalf("expected files %v, got %v", want, files)
	}
}
