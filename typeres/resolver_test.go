package typeres

import (
	"os"
	"path/filepath"
	"testing"
)

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

	resolver := NewResolver(dir)

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
