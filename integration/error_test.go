package integration

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"testing"

	di "github.com/asp24/gendi"
	"github.com/asp24/gendi/generator"
	"github.com/asp24/gendi/yaml"
)

// TestErrorCases tests that invalid configurations are properly rejected
func TestErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		expectError string
		phase       string // "load", "generate", or "compile"
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
			srcDir := path.Join("testdata", "errors", tt.name)
			tmpDir := prepareTestDir(t, srcDir)
			if err := prepareMainGo(srcDir, tmpDir); err != nil {
				t.Fatal(err)
			}

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
			opts := generator.Options{
				Package:    "main",
				ModuleRoot: tmpDir,
			}
			opts.Finalize()

			gen := generator.New(opts)
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

// TestInvalidImports tests various import error scenarios
func TestInvalidImports(t *testing.T) {
	tests := []struct {
		name        string
		files       map[string]string
		expectError string
	}{
		{
			name: "import_nonexistent_file",
			files: map[string]string{
				"gendi.yaml": `
imports:
  - ./nonexistent.yaml
`,
			},
			expectError: "import.*not found",
		},
		{
			name: "import_circular",
			files: map[string]string{
				"gendi.yaml": `
imports:
  - ./other.yaml
`,
				"other.yaml": `
imports:
  - ./gendi.yaml
`,
			},
			expectError: "circular",
		},
		{
			name: "import_invalid_yaml",
			files: map[string]string{
				"gendi.yaml": `
imports:
  - ./invalid.yaml
`,
				"invalid.yaml": `
this is not: valid: yaml: syntax:::
`,
			},
			expectError: "yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Write all files
			for name, content := range tt.files {
				path := filepath.Join(tmpDir, name)
				if err := os.WriteFile(path, []byte(content), 0644); err != nil {
					t.Fatalf("failed to write %s: %v", name, err)
				}
			}

			// Try to load config
			configPath := filepath.Join(tmpDir, "gendi.yaml")
			_, err := yaml.LoadConfig(configPath)

			if err == nil {
				t.Fatal("expected import error, got none")
			}

			matched, _ := regexp.MatchString("(?i)"+tt.expectError, err.Error())
			if !matched {
				t.Errorf("expected error matching %q, got: %v", tt.expectError, err)
			}
		})
	}
}
