package yaml

import (
	"path/filepath"
	"regexp"
	"runtime"
	"testing"
)

func getCurrentDir() string {
	_, filename, _, _ := runtime.Caller(0)

	return filepath.Dir(filename)
}

// TestInvalidImports tests various import error scenarios.
// These tests remain in integration/ because they test the import resolution
// logic which involves file system operations and is separate from generation errors.
func TestInvalidImports(t *testing.T) {
	tests := []struct {
		name        string
		expectError string
	}{
		{
			name:        "import_nonexistent_file",
			expectError: "import.*not found",
		},
		{
			name:        "import_circular",
			expectError: "circular",
		},
		{
			name:        "import_invalid_yaml",
			expectError: "yaml",
		},
	}

	currentDir := getCurrentDir()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := filepath.Join(currentDir, "testdata", tt.name, "gendi.yaml")

			_, err := LoadConfig(configPath)

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
