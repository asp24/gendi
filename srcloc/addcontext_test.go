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

func TestAddContext(t *testing.T) {
	loc := &Location{File: "/a.yaml", Line: 5, Column: 3}
	plain := errors.New("boom")
	cause := errors.New("cause")

	tests := []struct {
		name string
		// inErr is built per-row; some rows need to share an existing
		// *Error pointer to assert it's preserved unchanged.
		inErr  error
		format string
		args   []any
		// wantMsg is the expected Error() output.
		wantMsg string
		// wantLoc, when non-nil, must equal the Loc reachable via
		// errors.As(result, &*Error).
		wantLoc *Location
		// wantUnwrapTarget, when non-nil, must satisfy
		// errors.Is(result, wantUnwrapTarget).
		wantUnwrapTarget error
	}{
		{
			name:    "direct *Error folds prefix into Message and keeps Loc",
			inErr:   &Error{Loc: loc, Message: "bad value"},
			format:  "parameter %q",
			args:    []any{"foo"},
			wantMsg: `/a.yaml:5:3: parameter "foo": bad value`,
			wantLoc: loc,
		},
		{
			name:             "plain error falls back through fmt.Errorf with %w",
			inErr:            plain,
			format:           "context",
			wantMsg:          "context: boom",
			wantUnwrapTarget: plain,
		},
		{
			name: "wrapped *Error falls through; outer wrapper preserved",
			inErr: fmt.Errorf("outer: %w",
				&Error{Loc: loc, Message: "inner"}),
			format:  "prefix",
			wantMsg: "prefix: outer: /a.yaml:5:3: inner",
			wantLoc: loc, // still reachable via errors.As
		},
		{
			name:             "Err field on direct *Error is preserved through folding",
			inErr:            &Error{Loc: nil, Message: "wrapped", Err: cause},
			format:           "ctx",
			wantMsg:          "ctx: wrapped: cause",
			wantUnwrapTarget: cause,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AddContext(tt.inErr, tt.format, tt.args...)
			if got.Error() != tt.wantMsg {
				t.Errorf("Error() = %q, want %q", got.Error(), tt.wantMsg)
			}
			if tt.wantLoc != nil {
				var le *Error
				if !errors.As(got, &le) {
					t.Fatal("expected *Error reachable via errors.As")
				}
				if le.Loc != tt.wantLoc {
					t.Errorf("Loc = %v, want %v", le.Loc, tt.wantLoc)
				}
			}
			if tt.wantUnwrapTarget != nil && !errors.Is(got, tt.wantUnwrapTarget) {
				t.Error("errors.Is should find the unwrap target")
			}
		})
	}
}
