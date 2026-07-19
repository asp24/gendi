package imprt

import (
	"path/filepath"
	"testing"
)

// A glob that matches nothing is a silent no-op — whether its base directory
// exists or not. Only a malformed pattern is an error.
func TestGlobMatches(t *testing.T) {
	dir := t.TempDir()

	files, dirs, err := globMatches(dir + "/*.yaml")
	if err != nil {
		t.Fatalf("unexpected error for empty match: %v", err)
	}
	if len(files) != 0 || len(dirs) != 0 {
		t.Fatalf("expected no matches, got files=%v dirs=%v", files, dirs)
	}

	files, dirs, err = globMatches(dir + "/no_such_dir/*.yaml")
	if err != nil {
		t.Fatalf("unexpected error for missing base directory: %v", err)
	}
	if len(files) != 0 || len(dirs) != 0 {
		t.Fatalf("expected no matches, got files=%v dirs=%v", files, dirs)
	}

	if _, _, err = globMatches(dir + "/[invalid"); err == nil {
		t.Fatal("expected error for malformed pattern")
	}

	writeFile(t, filepath.Join(dir, "sub", "a.yaml"), "a")
	writeFile(t, filepath.Join(dir, "b.yaml"), "b")
	files, dirs, err = globMatches(dir + "/*")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantFiles := []string{mustAbs(t, filepath.Join(dir, "b.yaml"))}
	wantDirs := []string{mustAbs(t, filepath.Join(dir, "sub"))}
	if len(files) != 1 || files[0] != wantFiles[0] {
		t.Fatalf("expected files %v, got %v", wantFiles, files)
	}
	if len(dirs) != 1 || dirs[0] != wantDirs[0] {
		t.Fatalf("expected dirs %v, got %v", wantDirs, dirs)
	}
}
