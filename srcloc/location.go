package srcloc

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Location represents a position in a YAML source file.
type Location struct {
	File   string // Absolute path to the file
	Line   int    // 1-based line number
	Column int    // 1-based column number
}

func NewLocation(filePath string, node *yaml.Node) *Location {
	if filePath == "" || node == nil {
		return nil
	}

	return &Location{
		File:   filePath,
		Line:   node.Line,
		Column: node.Column,
	}
}

// String returns a formatted location string in the form "file:line:column".
func (l *Location) String() string {
	if l == nil {
		return ""
	}
	return fmt.Sprintf("%s:%d:%d", l.File, l.Line, l.Column)
}
