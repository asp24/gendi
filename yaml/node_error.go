package yaml

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// NodeError carries a yaml.Node for location tracking.
// It is produced during UnmarshalYAML and later enriched with a file path
// to become a srcloc.Error.
type NodeError struct {
	Node *yaml.Node
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

func nodeErrorf(node *yaml.Node, format string, args ...any) error {
	return &NodeError{
		Node: node,
		Msg:  fmt.Sprintf(format, args...),
	}
}

func wrapNodeError(node *yaml.Node, msg string, err error) error {
	return &NodeError{
		Node: node,
		Msg:  msg,
		Err:  err,
	}
}
