package imprt

import "testing"

func TestGlobFilesMissingBaseDirErrors(t *testing.T) {
	dir := t.TempDir()

	// Empty match inside an existing directory is a valid no-op.
	files, err := globFiles(dir + "/*.yaml")
	if err != nil {
		t.Fatalf("unexpected error for empty match: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("expected no files, got %v", files)
	}

	// A glob rooted at a non-existent directory is a typo, not a no-op.
	_, err = globFiles(dir + "/no_such_dir/*.yaml")
	if err == nil {
		t.Fatal("expected error for glob with missing base directory")
	}
}
