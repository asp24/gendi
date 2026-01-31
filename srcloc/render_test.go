package srcloc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderer_RenderLocation(t *testing.T) {
	// Create a test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.yaml")
	content := `services:
  logger:
    type: "*Logger"
    constructor:
      func: "NewLogger"
      args:
        - "@database"
        - "%log.level%"
  handler:
    type: "*Handler"`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	renderer := NewRenderer()

	tests := []struct {
		name         string
		loc          *Location
		contextLines int
		wantContains []string
	}{
		{
			name: "middle line with context",
			loc: &Location{
				File:   testFile,
				Line:   7,
				Column: 9,
			},
			contextLines: 2,
			wantContains: []string{
				"5 |       func: \"NewLogger\"",
				"6 |       args:",
				"7 |         - \"@database\"",
				"^",
				"8 |         - \"%log.level%\"",
				"9 |   handler:",
			},
		},
		{
			name: "first line",
			loc: &Location{
				File:   testFile,
				Line:   1,
				Column: 1,
			},
			contextLines: 1,
			wantContains: []string{
				"1 | services:",
				"^",
				"2 |   logger:",
			},
		},
		{
			name: "last line",
			loc: &Location{
				File:   testFile,
				Line:   10,
				Column: 5,
			},
			contextLines: 1,
			wantContains: []string{
				"9 |   handler:",
				"10 |     type: \"*Handler\"",
				"^",
			},
		},
		{
			name: "zero context",
			loc: &Location{
				File:   testFile,
				Line:   5,
				Column: 1,
			},
			contextLines: 0,
			wantContains: []string{
				"5 |       func: \"NewLogger\"",
				"^",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := renderer.RenderLocation(tt.loc, tt.contextLines)
			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("RenderLocation() missing %q\nGot:\n%s", want, got)
				}
			}
		})
	}
}

func TestRenderer_RenderLocation_EdgeCases(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.yaml")
	content := "line1\nline2\nline3\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	renderer := NewRenderer()

	tests := []struct {
		name string
		loc  *Location
		want string
	}{
		{
			name: "nil location",
			loc:  nil,
			want: "",
		},
		{
			name: "file not loaded",
			loc: &Location{
				File:   "/not/loaded.yaml",
				Line:   1,
				Column: 1,
			},
			want: "",
		},
		{
			name: "line out of range (too high)",
			loc: &Location{
				File:   testFile,
				Line:   100,
				Column: 1,
			},
			want: "",
		},
		{
			name: "line out of range (zero)",
			loc: &Location{
				File:   testFile,
				Line:   0,
				Column: 1,
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := renderer.RenderLocation(tt.loc, 2)
			if got != tt.want {
				t.Errorf("RenderLocation() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderer_RenderError(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.yaml")
	content := "services:\n  logger:\n    type: \"*Logger\"\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	renderer := NewRenderer()

	tests := []struct {
		name         string
		err          error
		wantContains []string
	}{
		{
			name: "error with location and cached file",
			err: &Error{
				Loc:     &Location{File: testFile, Line: 2, Column: 3},
				Message: "invalid service",
			},
			wantContains: []string{
				testFile + ":2:3: invalid service",
				"1 | services:",
				"2 |   logger:",
				"^",
				"3 |     type: \"*Logger\"",
			},
		},
		{
			name: "error without location",
			err: &Error{
				Message: "generic error",
			},
			wantContains: []string{
				"generic error",
			},
		},
		{
			name: "non-srcloc error",
			err:  os.ErrNotExist,
			wantContains: []string{
				"file does not exist",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderer.RenderError(tt.err, 1)
			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("GetErrorWithSnippet() missing %q\nGot:\n%s", want, got)
				}
			}
		})
	}
}
