package yaml

import (
	"github.com/goccy/go-yaml/ast"

	"github.com/asp24/gendi/srcloc"
)

// newLocation builds a *srcloc.Location from a goccy AST node.
// Returns nil if n is nil or has no token position.
func newLocation(filePath string, n ast.Node) *srcloc.Location {
	if n == nil {
		return nil
	}
	tok := n.GetToken()
	if tok == nil || tok.Position == nil {
		return nil
	}
	return srcloc.NewLocation(filePath, tok.Position.Line, tok.Position.Column)
}
