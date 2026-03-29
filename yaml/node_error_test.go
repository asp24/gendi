package yaml

import (
	"errors"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestNodeError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *NodeError
		want string
	}{
		{
			name: "message only",
			err:  &NodeError{Node: &yaml.Node{Line: 5, Column: 3}, Msg: "tag name is required"},
			want: "tag name is required",
		},
		{
			name: "message with wrapped error",
			err:  &NodeError{Node: &yaml.Node{Line: 5, Column: 3}, Msg: "failed to decode tag name", Err: errors.New("cannot unmarshal")},
			want: "failed to decode tag name: cannot unmarshal",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("NodeError.Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNodeError_Unwrap(t *testing.T) {
	inner := errors.New("inner")
	ne := &NodeError{Node: &yaml.Node{}, Msg: "outer", Err: inner}
	if got := ne.Unwrap(); got != inner {
		t.Errorf("NodeError.Unwrap() = %v, want %v", got, inner)
	}
}

func TestNodeError_ErrorsAs(t *testing.T) {
	ne := nodeErrorf(&yaml.Node{Line: 10, Column: 2}, "bad value")
	var target *NodeError
	if !errors.As(ne, &target) {
		t.Fatal("errors.As should find NodeError")
	}
	if target.Node.Line != 10 || target.Node.Column != 2 {
		t.Errorf("got line=%d col=%d, want line=10 col=2", target.Node.Line, target.Node.Column)
	}
}

func TestNodeErrorf(t *testing.T) {
	node := &yaml.Node{Line: 3, Column: 7}
	err := nodeErrorf(node, "service %q not found", "logger")
	ne := err.(*NodeError)
	if ne.Msg != `service "logger" not found` {
		t.Errorf("Msg = %q, want %q", ne.Msg, `service "logger" not found`)
	}
	if ne.Node != node {
		t.Error("Node should be the same pointer")
	}
	if ne.Err != nil {
		t.Error("Err should be nil")
	}
}

func TestWrapNodeError(t *testing.T) {
	node := &yaml.Node{Line: 1, Column: 1}
	inner := errors.New("decode failed")
	err := wrapNodeError(node, "failed to decode tag name", inner)
	ne := err.(*NodeError)
	if ne.Msg != "failed to decode tag name" {
		t.Errorf("Msg = %q", ne.Msg)
	}
	if ne.Err != inner {
		t.Error("Err should be inner")
	}
}
