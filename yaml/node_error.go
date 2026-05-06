package yaml

import (
	"fmt"

	"github.com/goccy/go-yaml/ast"
)

// NodeError carries a goccy ast.Node for location tracking.
// It is produced during UnmarshalYAML and later enriched with a file
// path to become a srcloc.Error inside ConfigLoaderYaml.toSrclocError.
type NodeError struct {
	Node ast.Node
	Msg  string
	Err  error
}

func (e *NodeError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Msg, e.Err)
	}
	return e.Msg
}

func (e *NodeError) Unwrap() error {
	return e.Err
}

func nodeErrorf(node ast.Node, format string, args ...any) error {
	return &NodeError{
		Node: node,
		Msg:  fmt.Sprintf(format, args...),
	}
}

func wrapNodeError(node ast.Node, msg string, err error) error {
	return &NodeError{
		Node: node,
		Msg:  msg,
		Err:  err,
	}
}
