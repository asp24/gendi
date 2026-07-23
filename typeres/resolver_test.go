package typeres

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLookupTypeComposite(t *testing.T) {
	// Composite branches recurse on their element type. Using universe
	// element types (int, string) keeps this a pure string-parsing test —
	// no package loading required.
	r := NewResolver("", "")

	tests := []struct {
		name string
		in   string
		want string
	}{
		{"array", "[3]int", "[3]int"},
		{"map", "map[string]int", "map[string]int"},
		{"receive-only channel", "<-chan int", "<-chan int"},
		{"send-only channel", "chan<- int", "chan<- int"},
		{"nested map key", "map[[2]int]string", "map[[2]int]string"},
		{"map of slice", "map[string][]int", "map[string][]int"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typ, err := r.LookupType(tt.in)
			if err != nil {
				t.Fatalf("LookupType(%q): %v", tt.in, err)
			}
			if got := typ.String(); got != tt.want {
				t.Fatalf("LookupType(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestLookupTypeCompositeErrors(t *testing.T) {
	r := NewResolver("", "")

	tests := []struct {
		name string
		in   string
	}{
		{"array missing bracket", "[3int"},
		{"array non-numeric size", "[abc]int"},
		{"array bad element", "[3]nope.Missing"},
		{"map missing bracket", "map[string"},
		{"map bad key", "map[nope.Missing]int"},
		{"map bad value", "map[string]nope.Missing"},
		{"channel bad element", "<-chan nope.Missing"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := r.LookupType(tt.in); err == nil {
				t.Fatalf("LookupType(%q): expected error, got nil", tt.in)
			}
		})
	}
}

func TestResolverUsesModuleRoot(t *testing.T) {
	dir := t.TempDir()
	modPath := "example.com/testmod"

	modFile := "module " + modPath + "\n\ngo 1.20\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(modFile), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	pkgDir := filepath.Join(dir, "foo")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("create package dir: %v", err)
	}

	src := `package foo

type Item struct{}

func New() *Item {
	return &Item{}
}
`
	if err := os.WriteFile(filepath.Join(pkgDir, "foo.go"), []byte(src), 0o644); err != nil {
		t.Fatalf("write package source: %v", err)
	}

	resolver := NewResolver(dir, "")

	if err := resolver.LoadPackages([]string{modPath + "/foo"}); err != nil {
		t.Fatalf("load packages: %v", err)
	}

	if _, err := resolver.LookupType(modPath + "/foo.Item"); err != nil {
		t.Fatalf("lookup type: %v", err)
	}
	if _, err := resolver.LookupFunc(modPath+"/foo", "New"); err != nil {
		t.Fatalf("lookup func: %v", err)
	}
}

func TestResolverPassesBuildTags(t *testing.T) {
	dir := t.TempDir()
	modPath := "example.com/testmod"

	modFile := "module " + modPath + "\n\ngo 1.20\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(modFile), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	pkgDir := filepath.Join(dir, "foo")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("create package dir: %v", err)
	}

	src := `//go:build sometag

package foo

type Item struct{}

func New() *Item {
	return &Item{}
}
`
	if err := os.WriteFile(filepath.Join(pkgDir, "foo.go"), []byte(src), 0o644); err != nil {
		t.Fatalf("write package source: %v", err)
	}

	t.Run("with tags", func(t *testing.T) {
		resolver := NewResolver(dir, "sometag")

		if err := resolver.LoadPackages([]string{modPath + "/foo"}); err != nil {
			t.Fatalf("load packages: %v", err)
		}

		if _, err := resolver.LookupType(modPath + "/foo.Item"); err != nil {
			t.Fatalf("lookup type: %v", err)
		}
		if _, err := resolver.LookupFunc(modPath+"/foo", "New"); err != nil {
			t.Fatalf("lookup func: %v", err)
		}
	})

	t.Run("without tags", func(t *testing.T) {
		resolver := NewResolver(dir, "")

		err := resolver.LoadPackages([]string{modPath + "/foo"})
		if err == nil {
			if _, err = resolver.LookupType(modPath + "/foo.Item"); err == nil {
				t.Fatal("expected tag-guarded type to be unresolvable without build tags")
			}
		}
	})
}
