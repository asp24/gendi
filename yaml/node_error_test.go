package yaml

import (
	"errors"
	"testing"
)

func TestNodeError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *NodeError
		want string
	}{
		{
			name: "message only",
			err:  &NodeError{Node: mustParseNode(t, "x"), Msg: "tag name is required"},
			want: "tag name is required",
		},
		{
			name: "message with wrapped error",
			err:  &NodeError{Node: mustParseNode(t, "x"), Msg: "failed to decode tag name", Err: errors.New("cannot unmarshal")},
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
	ne := &NodeError{Node: mustParseNode(t, "x"), Msg: "outer", Err: inner}
	if got := ne.Unwrap(); got != inner {
		t.Errorf("NodeError.Unwrap() = %v, want %v", got, inner)
	}
}

func TestNodeError_ErrorsAs(t *testing.T) {
	node := mustParseNode(t, "x")
	ne := nodeErrorf(node, "bad value")
	var target *NodeError
	if !errors.As(ne, &target) {
		t.Fatal("errors.As should find NodeError")
	}
	if target.Node != node {
		t.Errorf("Node not preserved")
	}
}

func TestNodeErrorf(t *testing.T) {
	node := mustParseNode(t, "x")
	err := nodeErrorf(node, "service %q not found", "logger")
	ne := err.(*NodeError)
	if ne.Msg != `service "logger" not found` {
		t.Errorf("Msg = %q, want %q", ne.Msg, `service "logger" not found`)
	}
	if ne.Node != node {
		t.Error("Node should be the same value")
	}
	if ne.Err != nil {
		t.Error("Err should be nil")
	}
}

func TestWrapNodeError(t *testing.T) {
	node := mustParseNode(t, "x")
	inner := errors.New("decode failed")
	err := wrapNodeError(node, "failed to decode tag name", inner)
	ne := err.(*NodeError)
	if ne.Msg != "failed to decode tag name" {
		t.Errorf("Msg = %q", ne.Msg)
	}
	if ne.Err != inner {
		t.Error("Err should be inner")
	}
	if ne.Node != node {
		t.Error("Node should be preserved")
	}
}

func TestRawImport_UnmarshalYAML_NodeError(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantMsg string
	}{
		{
			name:    "missing path in mapping",
			yaml:    "exclude:\n  - foo",
			wantMsg: "import path is required",
		},
		{
			name:    "wrong node type",
			yaml:    "[1, 2]",
			wantMsg: "import must be a string or mapping",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var imp RawImport
			err := imp.UnmarshalYAML(mustParseNode(t, tt.yaml))

			var ne *NodeError
			if !errors.As(err, &ne) {
				t.Fatalf("expected NodeError, got %T: %v", err, err)
			}
			if ne.Msg != tt.wantMsg {
				t.Errorf("Msg = %q, want %q", ne.Msg, tt.wantMsg)
			}
			if ne.Node == nil {
				t.Error("Node should be non-nil")
			}
		})
	}
}

func TestRawService_UnmarshalYAML_NodeError(t *testing.T) {
	var svc RawService
	err := svc.UnmarshalYAML(mustParseNode(t, "[1, 2]"))

	var ne *NodeError
	if !errors.As(err, &ne) {
		t.Fatalf("expected NodeError, got %T: %v", err, err)
	}
	if ne.Msg != "service must be a mapping or alias" {
		t.Errorf("Msg = %q", ne.Msg)
	}
}

func TestRawServiceTag_UnmarshalYAML_NodeError(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantMsg string
	}{
		{
			name:    "empty scalar name",
			yaml:    `""`,
			wantMsg: "tag name is required",
		},
		{
			name:    "wrong node type",
			yaml:    "[1, 2]",
			wantMsg: "tag must be a string or mapping",
		},
		{
			name:    "missing name in mapping",
			yaml:    "priority: 10",
			wantMsg: "tag name is required",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tag RawServiceTag
			err := tag.UnmarshalYAML(mustParseNode(t, tt.yaml))

			var ne *NodeError
			if !errors.As(err, &ne) {
				t.Fatalf("expected NodeError, got %T: %v", err, err)
			}
			if ne.Msg != tt.wantMsg {
				t.Errorf("Msg = %q, want %q", ne.Msg, tt.wantMsg)
			}
			if ne.Node == nil {
				t.Error("Node should be non-nil")
			}
		})
	}
}
