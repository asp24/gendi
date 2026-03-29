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

func TestRawImport_UnmarshalYAML_NodeError(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		wantMsg  string
		wantLine int
	}{
		{
			name:     "missing path in mapping",
			yaml:     "exclude:\n  - foo",
			wantMsg:  "import path is required",
			wantLine: 1,
		},
		{
			name:     "wrong node type",
			yaml:     "[1, 2]",
			wantMsg:  "import must be a string or mapping",
			wantLine: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var node yaml.Node
			if err := yaml.Unmarshal([]byte(tt.yaml), &node); err != nil {
				t.Fatal(err)
			}
			var imp RawImport
			err := imp.UnmarshalYAML(node.Content[0])

			var ne *NodeError
			if !errors.As(err, &ne) {
				t.Fatalf("expected NodeError, got %T: %v", err, err)
			}
			if ne.Msg != tt.wantMsg {
				t.Errorf("Msg = %q, want %q", ne.Msg, tt.wantMsg)
			}
			if ne.Node.Line != tt.wantLine {
				t.Errorf("Line = %d, want %d", ne.Node.Line, tt.wantLine)
			}
		})
	}
}

func TestRawService_UnmarshalYAML_NodeError(t *testing.T) {
	yamlInput := "[1, 2]"
	var node yaml.Node
	if err := yaml.Unmarshal([]byte(yamlInput), &node); err != nil {
		t.Fatal(err)
	}
	var svc RawService
	err := svc.UnmarshalYAML(node.Content[0])

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
		name     string
		yaml     string
		wantMsg  string
		wantLine int
	}{
		{
			name:     "empty scalar name",
			yaml:     `""`,
			wantMsg:  "tag name is required",
			wantLine: 1,
		},
		{
			name:     "wrong node type",
			yaml:     "[1, 2]",
			wantMsg:  "tag must be a string or mapping",
			wantLine: 1,
		},
		{
			name:     "missing name in mapping",
			yaml:     "priority: 10",
			wantMsg:  "tag name is required",
			wantLine: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var node yaml.Node
			if err := yaml.Unmarshal([]byte(tt.yaml), &node); err != nil {
				t.Fatal(err)
			}
			var tag RawServiceTag
			err := tag.UnmarshalYAML(node.Content[0])

			var ne *NodeError
			if !errors.As(err, &ne) {
				t.Fatalf("expected NodeError, got %T: %v", err, err)
			}
			if ne.Msg != tt.wantMsg {
				t.Errorf("Msg = %q, want %q", ne.Msg, tt.wantMsg)
			}
			if ne.Node.Line != tt.wantLine {
				t.Errorf("Line = %d, want %d", ne.Node.Line, tt.wantLine)
			}
		})
	}
}
