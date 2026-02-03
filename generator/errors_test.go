package generator

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	di "github.com/asp24/gendi"
	"github.com/asp24/gendi/yaml"
)

// TestErrorCases tests that invalid configurations are properly rejected during the
// YAML loading and code generation pipeline (not including compilation/runtime).
func TestErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		expectError string
		phase       string // "load" or "generate"
	}{
		{
			name:        "circular_dependency",
			expectError: "circular",
			phase:       "generate",
		},
		{
			name:        "missing_service",
			expectError: "not found",
			phase:       "generate",
		},
		{
			name:        "missing_parameter",
			expectError: "not found",
			phase:       "generate",
		},
		{
			name:        "invalid_parameter_type",
			expectError: "type",
			phase:       "load",
		},
		{
			name:        "missing_constructor",
			expectError: "constructor",
			phase:       "generate",
		},
		{
			name:        "both_func_and_method",
			expectError: "both func and method",
			phase:       "generate",
		},
		{
			name:        "decorator_missing_inner",
			expectError: "inner",
			phase:       "generate",
		},
		{
			name:        "spread_not_last",
			expectError: "spread.*last",
			phase:       "generate",
		},
		{
			name:        "autoconfigure_with_sortby",
			expectError: "autoconfigure.*sort_by",
			phase:       "generate",
		},
		{
			name:        "invalid_alias",
			expectError: "alias.*not found",
			phase:       "generate",
		},
		{
			name:        "alias_with_constructor",
			expectError: "alias.*constructor",
			phase:       "generate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srcDir := filepath.Join("testdata", "errors", tt.name)
			tmpDir := prepareErrorTestDir(t, srcDir)

			configPath := filepath.Join(tmpDir, "gendi.yaml")
			cfg, err := yaml.LoadConfig(configPath)

			if tt.phase == "load" {
				if err == nil {
					t.Fatal("expected error during load, got none")
				}
				matched, _ := regexp.MatchString("(?i)"+tt.expectError, err.Error())
				if !matched {
					t.Errorf("expected error matching %q, got: %v", tt.expectError, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected load error: %v", err)
			}

			// Apply internal passes
			cfg, err = di.ApplyInternalPasses(cfg)
			if err != nil {
				t.Fatalf("unexpected pass error: %v", err)
			}

			// Generate
			opts := Options{
				Package:    "main",
				ModuleRoot: tmpDir,
			}
			opts.Finalize()

			gen := New(opts)
			_, err = gen.Generate(cfg)

			if tt.phase == "generate" {
				if err == nil {
					t.Fatal("expected error during generation, got none")
				}
				matched, _ := regexp.MatchString("(?i)"+tt.expectError, err.Error())
				if !matched {
					t.Errorf("expected error matching %q, got: %v", tt.expectError, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected generation error: %v", err)
			}
		})
	}
}

// prepareErrorTestDir creates a temporary test directory with go.mod for error tests
func prepareErrorTestDir(t *testing.T, srcDir string) string {
	t.Helper()

	tmpDir := t.TempDir()

	// Copy files from srcDir to tmpDir
	if err := copyDir(srcDir, tmpDir, nil); err != nil {
		t.Fatal(err)
	}

	// Create go.mod
	goModPath := filepath.Join(tmpDir, "go.mod")
	moduleRoot, err := getModuleRoot()
	if err != nil {
		t.Fatalf("get module root: %v", err)
	}

	goModContent := fmt.Sprintf(`module test

go 1.25.4

require github.com/asp24/gendi v0.0.0

replace github.com/asp24/gendi => %s
`, moduleRoot)

	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a stub main.go if it doesn't exist
	mainGoPath := filepath.Join(tmpDir, "main.go")
	if _, err := os.Stat(mainGoPath); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(mainGoPath, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	return tmpDir
}

// copyDir recursively copies a directory
func copyDir(src, dst string, exclude []string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		// Check if file is excluded
		for _, ex := range exclude {
			if entry.Name() == ex {
				continue
			}
		}

		if entry.IsDir() {
			if err := os.MkdirAll(dstPath, 0755); err != nil {
				return err
			}
			if err := copyDir(srcPath, dstPath, exclude); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	return nil
}

// copyFile copies a single file
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

// getModuleRoot returns the root directory of the gendi module
func getModuleRoot() (string, error) {
	// Walk up from current directory to find go.mod
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found")
		}
		dir = parent
	}
}
