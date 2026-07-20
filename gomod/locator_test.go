package gomod

import (
	"os"
	"path/filepath"
	"testing"
)

// A relative startDir walks the ancestors of its absolute location — the
// relative spelling alone would stop at the working directory. No t.Parallel
// — t.Chdir forbids it.
func TestFindModuleRootRelativeDir(t *testing.T) {
	moduleRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(moduleRoot, "go.mod"), []byte("module example.com/app\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	sub := filepath.Join(moduleRoot, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	t.Chdir(sub)

	dir, modPath, found := FindModuleRoot(".")
	if !found {
		t.Fatal("expected module root to be found from a relative startDir")
	}
	if modPath != "example.com/app" {
		t.Fatalf("module path = %q, want %q", modPath, "example.com/app")
	}
	realWant, err := filepath.EvalSymlinks(moduleRoot)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	realGot, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	if realGot != realWant {
		t.Fatalf("module root = %q, want %q", dir, moduleRoot)
	}
}

func TestParseModulePath(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "module example.com/m\n", "example.com/m"},
		{"trailing comment", "module example.com/m // main module\n", "example.com/m"},
		{"quoted", "module \"example.com/m\"\n", "example.com/m"},
		{"backquoted", "module `example.com/m`\n", "example.com/m"},
		{"after go directive", "go 1.22\nmodule example.com/m\n", "example.com/m"},
		{"missing", "go 1.22\n", ""},
		{"commented out", "// module example.com/other\nmodule example.com/m\n", "example.com/m"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseModulePath([]byte(tt.in)); got != tt.want {
				t.Errorf("ParseModulePath(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
