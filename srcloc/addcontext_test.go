package srcloc

import (
	"errors"
	"fmt"
	"testing"
)

func TestAddContext_NilInput(t *testing.T) {
	if got := AddContext(nil, "anything"); got != nil {
		t.Errorf("AddContext(nil, ...) = %v, want nil", got)
	}
}

func TestAddContext_DirectLocatedError_FoldsPrefix(t *testing.T) {
	loc := &Location{File: "/a/b.yaml", Line: 5, Column: 3}
	original := &Error{Loc: loc, Message: "bad value"}

	got := AddContext(original, "parameter %q", "foo")

	want := `/a/b.yaml:5:3: parameter "foo": bad value`
	if got.Error() != want {
		t.Errorf("Error() = %q, want %q", got.Error(), want)
	}
	var le *Error
	if !errors.As(got, &le) {
		t.Fatal("expected *Error in chain")
	}
	if le.Loc != loc {
		t.Errorf("Loc not preserved")
	}
}

func TestAddContext_PlainError_FallsBack(t *testing.T) {
	original := errors.New("boom")

	got := AddContext(original, "context")

	if got.Error() != "context: boom" {
		t.Errorf("Error() = %q, want %q", got.Error(), "context: boom")
	}
	if !errors.Is(got, original) {
		t.Error("errors.Is should find original")
	}
}

func TestAddContext_WrappedLocatedError_DoesNotFoldOuter(t *testing.T) {
	loc := &Location{File: "/a.yaml", Line: 1, Column: 1}
	inner := &Error{Loc: loc, Message: "inner"}
	wrapped := fmt.Errorf("outer: %w", inner)

	got := AddContext(wrapped, "prefix")

	// "outer" must be preserved — not silently discarded.
	if got.Error() != "prefix: outer: /a.yaml:1:1: inner" {
		t.Errorf("Error() = %q", got.Error())
	}
	// Inner *Error still findable by errors.As (so Renderer can find Loc).
	var le *Error
	if !errors.As(got, &le) {
		t.Fatal("expected *Error reachable via errors.As")
	}
	if le.Loc != loc {
		t.Errorf("expected inner Loc, got %v", le.Loc)
	}
}

func TestAddContext_PreservesInnerErr(t *testing.T) {
	cause := errors.New("cause")
	original := &Error{Loc: nil, Message: "wrapped", Err: cause}

	got := AddContext(original, "ctx")

	if !errors.Is(got, cause) {
		t.Error("errors.Is should still find cause")
	}
	if got.Error() != "ctx: wrapped: cause" {
		t.Errorf("Error() = %q", got.Error())
	}
}
