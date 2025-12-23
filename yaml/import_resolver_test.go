package yaml

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestImportResolverResolveErrors(t *testing.T) {
	t.Parallel()

	resolver := NewImportResolver()
	tempDir := t.TempDir()

	_, err := resolver.Resolve(tempDir, "")
	if err == nil || !strings.Contains(err.Error(), "import path is empty") {
		t.Fatalf("expected empty import path error, got %v", err)
	}

	_, err = resolver.Resolve(tempDir, "./missing.yaml")
	if err == nil || !strings.Contains(err.Error(), "import not found at") {
		t.Fatalf("expected explicit relative missing error, got %v", err)
	}

	_, err = resolver.Resolve(tempDir, "missing")
	if err == nil || !strings.Contains(err.Error(), "import not found at") {
		t.Fatalf("expected non-module missing error, got %v", err)
	}
}

func TestImportResolverResolveLocalAndAbsolute(t *testing.T) {
	t.Parallel()

	resolver := NewImportResolver()
	tempDir := t.TempDir()

	relativePath := filepath.Join(tempDir, "config.yaml")
	writeFile(t, relativePath, "content")

	result, err := resolver.Resolve(tempDir, "config.yaml")
	if err != nil {
		t.Fatalf("resolve relative failed: %v", err)
	}
	expected := []string{mustAbs(t, relativePath)}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}

	absolutePath := filepath.Join(tempDir, "absolute.yaml")
	writeFile(t, absolutePath, "content")
	result, err = resolver.Resolve(tempDir, absolutePath)
	if err != nil {
		t.Fatalf("resolve absolute failed: %v", err)
	}
	expected = []string{mustAbs(t, absolutePath)}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestImportResolverResolveGlob(t *testing.T) {
	t.Parallel()

	resolver := NewImportResolver()
	tempDir := t.TempDir()

	writeFile(t, filepath.Join(tempDir, "a.yaml"), "a")
	writeFile(t, filepath.Join(tempDir, "b.yaml"), "b")
	writeFile(t, filepath.Join(tempDir, "c.txt"), "c")

	result, err := resolver.Resolve(tempDir, "./*.yaml")
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

func TestImportResolverResolveModuleDefaultConfig(t *testing.T) {
	t.Parallel()

	resolver := NewImportResolver()
	moduleRoot, baseDir, modulePath := createModule(t)
	defaultConfig := filepath.Join(moduleRoot, "gendi.yaml")
	writeFile(t, defaultConfig, "name: module")

	result, err := resolver.Resolve(baseDir, modulePath)
	if err != nil {
		t.Fatalf("resolve module default failed: %v", err)
	}
	expected := []string{mustAbs(t, defaultConfig)}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestImportResolverResolveModuleFile(t *testing.T) {
	t.Parallel()

	resolver := NewImportResolver()
	moduleRoot, baseDir, modulePath := createModule(t)
	configPath := filepath.Join(moduleRoot, "configs", "app.yaml")
	writeFile(t, configPath, "name: app")

	result, err := resolver.Resolve(baseDir, modulePath+"/configs/app.yaml")
	if err != nil {
		t.Fatalf("resolve module file failed: %v", err)
	}
	expected := []string{mustAbs(t, configPath)}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestImportResolverResolveModuleFileMissing(t *testing.T) {
	t.Parallel()

	resolver := NewImportResolver()
	_, baseDir, modulePath := createModule(t)

	_, err := resolver.Resolve(baseDir, modulePath+"/configs/missing.yaml")
	if err == nil || !strings.Contains(err.Error(), "does not contain") {
		t.Fatalf("expected missing module file error, got %v", err)
	}
}

func TestImportResolverResolveModuleGlob(t *testing.T) {
	t.Parallel()

	resolver := NewImportResolver()
	moduleRoot, baseDir, modulePath := createModule(t)
	writeFile(t, filepath.Join(moduleRoot, "configs", "a.yaml"), "a")
	writeFile(t, filepath.Join(moduleRoot, "configs", "b.yaml"), "b")
	writeFile(t, filepath.Join(moduleRoot, "configs", "c.txt"), "c")

	result, err := resolver.Resolve(baseDir, modulePath+"/configs/*.yaml")
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
