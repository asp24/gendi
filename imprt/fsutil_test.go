package imprt

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// A glob over an existing directory that matches nothing is a silent no-op; a
// glob whose base directory does not exist, or a malformed pattern, is an error.
func TestGlobMatches(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		setup   func(t *testing.T, root string)
		want    []string
		wantErr string
	}{
		{
			name:    "empty match",
			pattern: "*.yaml",
		},
		{
			name:    "missing base directory",
			pattern: "no_such_dir/*.yaml",
			wantErr: "does not exist",
		},
		{
			name:    "malformed pattern",
			pattern: "[invalid",
			wantErr: "glob",
		},
		{
			name:    "directories are excluded",
			pattern: "*",
			setup: func(t *testing.T, root string) {
				writeFile(t, filepath.Join(root, "sub", "a.yaml"), "a")
				writeFile(t, filepath.Join(root, "b.yaml"), "b")
			},
			want: []string{"b.yaml"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			if tt.setup != nil {
				tt.setup(t, root)
			}

			files, err := globMatches(root, tt.pattern)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			want := make([]string, len(tt.want))
			for i, path := range tt.want {
				want[i] = mustAbs(t, filepath.Join(root, path))
			}
			if !slices.Equal(files, want) {
				t.Fatalf("expected files %v, got %v", want, files)
			}
		})
	}
}

// Glob metacharacters in the anchor directory are literal path bytes, not
// pattern syntax.
func TestGlobMatchesMetacharInRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "work[2024]", "app")
	writeFile(t, filepath.Join(root, "services", "a.yaml"), "a")
	writeFile(t, filepath.Join(root, "services", "b.yaml"), "b")

	files, err := globMatches(root, "services/*.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{
		mustAbs(t, filepath.Join(root, "services", "a.yaml")),
		mustAbs(t, filepath.Join(root, "services", "b.yaml")),
	}
	if len(files) != 2 || files[0] != want[0] || files[1] != want[1] {
		t.Fatalf("expected files %v, got %v", want, files)
	}
}

// Native path separators in a glob are normalized before doublestar parses
// the pattern. In particular, Windows separators must not be read as escapes.
func TestGlobMatchesNativeSeparators(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "services", "a.yaml"), "a")
	writeFile(t, filepath.Join(root, "services", "nested", "b.yaml"), "b")

	files, err := globMatches(root, filepath.Join("services", "**", "*.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{
		mustAbs(t, filepath.Join(root, "services", "a.yaml")),
		mustAbs(t, filepath.Join(root, "services", "nested", "b.yaml")),
	}
	if len(files) != 2 || files[0] != want[0] || files[1] != want[1] {
		t.Fatalf("expected files %v, got %v", want, files)
	}
}
