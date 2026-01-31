package srcloc

import (
	"errors"
	"testing"
)

func TestError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *Error
		want string
	}{
		{
			name: "location and message only",
			err: &Error{
				Loc:     &Location{File: "file.yaml", Line: 10, Column: 5},
				Message: "invalid service",
			},
			want: "file.yaml:10:5: invalid service",
		},
		{
			name: "location, message, and wrapped error",
			err: &Error{
				Loc:     &Location{File: "file.yaml", Line: 10, Column: 5},
				Message: "invalid service",
				Err:     errors.New("unknown type"),
			},
			want: "file.yaml:10:5: invalid service: unknown type",
		},
		{
			name: "message only (no location)",
			err: &Error{
				Loc:     nil,
				Message: "invalid service",
			},
			want: "invalid service",
		},
		{
			name: "message and wrapped error (no location)",
			err: &Error{
				Loc:     nil,
				Message: "invalid service",
				Err:     errors.New("unknown type"),
			},
			want: "invalid service: unknown type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.want {
				t.Errorf("Error.Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestError_Unwrap(t *testing.T) {
	innerErr := errors.New("inner error")
	err := &Error{
		Loc:     &Location{File: "file.yaml", Line: 1, Column: 1},
		Message: "outer error",
		Err:     innerErr,
	}

	unwrapped := err.Unwrap()
	if unwrapped != innerErr {
		t.Errorf("Error.Unwrap() = %v, want %v", unwrapped, innerErr)
	}
}

func TestErrorf(t *testing.T) {
	tests := []struct {
		name   string
		loc    *Location
		format string
		args   []interface{}
		want   string
	}{
		{
			name:   "with location",
			loc:    &Location{File: "config.yaml", Line: 20, Column: 15},
			format: "service %q not found",
			args:   []interface{}{"logger"},
			want:   "config.yaml:20:15: service \"logger\" not found",
		},
		{
			name:   "without location",
			loc:    nil,
			format: "service %q not found",
			args:   []interface{}{"logger"},
			want:   "service \"logger\" not found",
		},
		{
			name:   "no args",
			loc:    &Location{File: "test.yaml", Line: 1, Column: 1},
			format: "error occurred",
			args:   nil,
			want:   "test.yaml:1:1: error occurred",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Errorf(tt.loc, tt.format, tt.args...)
			got := err.Error()
			if got != tt.want {
				t.Errorf("Errorf() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWrapError(t *testing.T) {
	innerErr := errors.New("inner error")

	tests := []struct {
		name string
		loc  *Location
		msg  string
		err  error
		want string
	}{
		{
			name: "with location",
			loc:  &Location{File: "app.yaml", Line: 5, Column: 3},
			msg:  "failed to parse",
			err:  innerErr,
			want: "app.yaml:5:3: failed to parse: inner error",
		},
		{
			name: "without location",
			loc:  nil,
			msg:  "failed to parse",
			err:  innerErr,
			want: "failed to parse: inner error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := WrapError(tt.loc, tt.msg, tt.err)
			got := err.Error()
			if got != tt.want {
				t.Errorf("WrapError() = %q, want %q", got, tt.want)
			}

			// Verify unwrapping works
			var wrapped *Error
			if errors.As(err, &wrapped) {
				if !errors.Is(wrapped.Unwrap(), innerErr) {
					t.Errorf("WrapError().Unwrap() = %v, want %v", wrapped.Unwrap(), innerErr)
				}
			}
		})
	}
}
