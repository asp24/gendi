package srcloc

import "fmt"

// AddContext prepends a contextual prefix to err's message without
// adding another location.
//
// If err is *directly* a *Error (no wrappers), returns a new *Error
// with the same Loc and Err, and Message = "<prefix>: <old message>".
// Otherwise wraps err with fmt.Errorf("<prefix>: %w", err) — preserving
// any outer wrappers verbatim.
//
// IMPORTANT: this uses a direct type assertion, NOT errors.As, so an
// already-wrapped located error (e.g. fmt.Errorf("outer: %w", locErr))
// goes through the fallback path. Using errors.As here would silently
// drop the "outer" wrapper.
func AddContext(err error, format string, args ...any) error {
	if err == nil {
		return nil
	}
	prefix := fmt.Sprintf(format, args...)
	if le, ok := err.(*Error); ok {
		return &Error{
			Loc:     le.Loc,
			Message: prefix + ": " + le.Message,
			Err:     le.Err,
		}
	}
	return fmt.Errorf("%s: %w", prefix, err)
}
