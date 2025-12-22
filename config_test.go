package di

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigImportPrefix(t *testing.T) {
	dir := t.TempDir()
	basePath := filepath.Join(dir, "base.yaml")
	rootPath := filepath.Join(dir, "root.yaml")

	base := []byte(strings.TrimSpace(`
services:
  base:
    constructor:
      func: "example.NewBase"
  api:
    constructor:
      func: "example.NewAPI"
      args:
        - "@base"
    decorates: "base"
`))
	if err := os.WriteFile(basePath, base, 0o644); err != nil {
		t.Fatalf("write base config: %v", err)
	}

	root := []byte(strings.TrimSpace(fmt.Sprintf(`
imports:
  - path: %q
    prefix: "mod."
`, "./base.yaml")))
	if err := os.WriteFile(rootPath, root, 0o644); err != nil {
		t.Fatalf("write root config: %v", err)
	}

	cfg, err := LoadConfig(rootPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if _, ok := cfg.Services["mod.base"]; !ok {
		t.Fatalf("expected prefixed service mod.base")
	}
	api := cfg.Services["mod.api"]
	if api == nil {
		t.Fatalf("expected prefixed service mod.api")
	}
	if got := api.Constructor.Args[0].Value; got != "mod.base" {
		t.Fatalf("expected service ref to be prefixed, got %q", got)
	}
	if api.Decorates != "mod.base" {
		t.Fatalf("expected decorates to be prefixed, got %q", api.Decorates)
	}
}

func TestLoadConfigModuleImport(t *testing.T) {
	modulePath := readModulePath(t)
	importPath := modulePath + "/internal/generator/testdata/imports/module.yaml"

	dir, err := os.MkdirTemp(".", "config-import-")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})

	rootPath := filepath.Join(dir, "root.yaml")
	root := []byte(strings.TrimSpace(fmt.Sprintf(`
imports:
  - path: %q
`, importPath)))
	if err := os.WriteFile(rootPath, root, 0o644); err != nil {
		t.Fatalf("write root config: %v", err)
	}

	cfg, err := LoadConfig(rootPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if _, ok := cfg.Services["module.service"]; !ok {
		t.Fatalf("expected service from module import to load")
	}
}

func TestLoadConfigImportGlobLocal(t *testing.T) {
	dir := t.TempDir()
	baseAPath := filepath.Join(dir, "base_a.yaml")
	baseBPath := filepath.Join(dir, "base_b.yaml")
	rootPath := filepath.Join(dir, "root.yaml")

	baseA := []byte(strings.TrimSpace(`
services:
  dupe:
    constructor:
      func: "example.NewA"
`))
	if err := os.WriteFile(baseAPath, baseA, 0o644); err != nil {
		t.Fatalf("write base_a config: %v", err)
	}

	baseB := []byte(strings.TrimSpace(`
services:
  dupe:
    constructor:
      func: "example.NewB"
  extra:
    constructor:
      func: "example.NewExtra"
`))
	if err := os.WriteFile(baseBPath, baseB, 0o644); err != nil {
		t.Fatalf("write base_b config: %v", err)
	}

	root := []byte(strings.TrimSpace(`
imports:
  - "./base_*.yaml"
`))
	if err := os.WriteFile(rootPath, root, 0o644); err != nil {
		t.Fatalf("write root config: %v", err)
	}

	cfg, err := LoadConfig(rootPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	dupe := cfg.Services["dupe"]
	if dupe == nil || dupe.Constructor.Func != "example.NewB" {
		t.Fatalf("expected dupe service to come from base_b")
	}
	if _, ok := cfg.Services["extra"]; !ok {
		t.Fatalf("expected extra service from base_b")
	}
}

func TestLoadConfigImportGlobModule(t *testing.T) {
	modulePath := readModulePath(t)
	importPath := modulePath + "/internal/generator/testdata/imports/*.yaml"

	dir := t.TempDir()
	rootPath := filepath.Join(dir, "root.yaml")
	root := []byte(strings.TrimSpace(fmt.Sprintf(`
imports:
  - path: %q
`, importPath)))
	if err := os.WriteFile(rootPath, root, 0o644); err != nil {
		t.Fatalf("write root config: %v", err)
	}

	cfg, err := LoadConfig(rootPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if _, ok := cfg.Services["module.service"]; !ok {
		t.Fatalf("expected service from module glob import to load")
	}
	if _, ok := cfg.Services["module.extra"]; !ok {
		t.Fatalf("expected extra service from module glob import to load")
	}
}

func readModulePath(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	t.Fatalf("module path not found in go.mod")
	return ""
}
