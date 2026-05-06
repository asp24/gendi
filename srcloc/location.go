package srcloc

import "fmt"

// Location represents a position in a YAML source file.
type Location struct {
	File   string // Absolute path to the file
	Line   int    // 1-based line number
	Column int    // 1-based column number
}

// NewLocation constructs a Location. Returns nil for empty file or
// non-positive line.
func NewLocation(filePath string, line, column int) *Location {
	if filePath == "" || line < 1 {
		return nil
	}
	return &Location{File: filePath, Line: line, Column: column}
}

// String returns a formatted location string in the form "file:line:column".
func (l *Location) String() string {
	if l == nil {
		return ""
	}
	return fmt.Sprintf("%s:%d:%d", l.File, l.Line, l.Column)
}
