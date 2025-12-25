package yaml

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePackagePath(t *testing.T) {
	// This test uses the actual project go.mod
	// Get the yaml package directory
	yamlDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	pkg, err := resolvePackagePath(yamlDir)
	if err != nil {
		t.Fatalf("resolvePackagePath failed: %v", err)
	}

	// Should end with /yaml since this is the yaml package
	if pkg != "github.com/asp24/gendi/yaml" {
		t.Errorf("expected package 'github.com/asp24/gendi/yaml', got '%s'", pkg)
	}
}

func TestResolvePackagePathModuleRoot(t *testing.T) {
	// Test with module root directory
	yamlDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	// Go up one level to module root
	moduleRoot := filepath.Dir(yamlDir)

	pkg, err := resolvePackagePath(moduleRoot)
	if err != nil {
		t.Fatalf("resolvePackagePath failed: %v", err)
	}

	// Module root should return just the module path
	if pkg != "github.com/asp24/gendi" {
		t.Errorf("expected package 'github.com/asp24/gendi', got '%s'", pkg)
	}
}

func TestResolvePackagePathNotFound(t *testing.T) {
	// Test with a directory that doesn't have go.mod
	// Use a temp directory
	tempDir := t.TempDir()

	_, err := resolvePackagePath(tempDir)
	if err == nil {
		t.Error("expected error when go.mod not found, got nil")
	}
}
