package yaml

import (
	"os"
	"path/filepath"
	"testing"
)

type stubResolver struct {
	paths map[string][]string
}

func (r stubResolver) CanResolve(string) bool { return true }

func (r stubResolver) Resolve(_, importPath string) ([]string, error) {
	if paths, ok := r.paths[importPath]; ok {
		return paths, nil
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

func TestLoadAllowsDiamondImports(t *testing.T) {
	dir := t.TempDir()

	rootPath := writeFile(t, dir, "root.yaml", `
imports:
  - path: b
  - path: c
parameters:
  a:
    type: string
    value: "A"
`)
	bPath := writeFile(t, dir, "b.yaml", `
imports:
  - path: d
parameters:
  b:
    type: string
    value: "B"
`)
	cPath := writeFile(t, dir, "c.yaml", `
imports:
  - path: d
parameters:
  c:
    type: string
    value: "C"
`)
	dPath := writeFile(t, dir, "d.yaml", `
parameters:
  d:
    type: string
    value: "D"
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

	cfg, err := loader.Load(rootPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got, want := len(cfg.Parameters), 4; got != want {
		t.Fatalf("expected %d parameters, got %d", want, got)
	}
	if readCount != 4 {
		t.Fatalf("expected each file read once, got %d reads", readCount)
	}
}
