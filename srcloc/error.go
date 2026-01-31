package srcloc

import (
	"fmt"
)

// Error wraps an error with source location information.
type Error struct {
	Loc     *Location
	Message string
	Err     error
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Loc != nil {
		if e.Err != nil {
			return fmt.Sprintf("%s: %s: %v", e.Loc, e.Message, e.Err)
		}
		return fmt.Sprintf("%s: %s", e.Loc, e.Message)
	}
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap returns the underlying error.
func (e *Error) Unwrap() error {
	return e.Err
}

// Errorf creates a new error with location information.
// If loc is nil, creates a regular formatted error.
func Errorf(loc *Location, format string, args ...interface{}) error {
	return &Error{
		Loc:     loc,
		Message: fmt.Sprintf(format, args...),
	}
}

// WrapError wraps an existing error with location information and a message.
// If loc is nil, creates a wrapped error without location.
func WrapError(loc *Location, msg string, err error) error {
	return &Error{
		Loc:     loc,
		Message: msg,
		Err:     err,
	}
}
