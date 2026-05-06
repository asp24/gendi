package srcloc

import "testing"

func TestNewLocation(t *testing.T) {
	loc := NewLocation("/a/b.yaml", 5, 3)
	if loc == nil {
		t.Fatal("expected non-nil")
	}
	if loc.File != "/a/b.yaml" || loc.Line != 5 || loc.Column != 3 {
		t.Errorf("unexpected location: %+v", loc)
	}
}

func TestNewLocation_EmptyFile_ReturnsNil(t *testing.T) {
	if NewLocation("", 5, 3) != nil {
		t.Error("expected nil for empty file")
	}
}

func TestNewLocation_ZeroLine_ReturnsNil(t *testing.T) {
	if NewLocation("/a.yaml", 0, 1) != nil {
		t.Error("expected nil for zero line")
	}
}

func TestLocation_String(t *testing.T) {
	loc := NewLocation("/a.yaml", 5, 3)
	if got := loc.String(); got != "/a.yaml:5:3" {
		t.Errorf("String() = %q", got)
	}
	var nilLoc *Location
	if got := nilLoc.String(); got != "" {
		t.Errorf("nil String() = %q", got)
	}
}
