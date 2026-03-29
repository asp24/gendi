# YAML Unmarshal Error Locations Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wrap errors from `UnmarshalYAML` methods with YAML node location info so they render with file:line:column and code snippets via the existing `srcloc.Renderer.RenderError` pipeline.

**Architecture:** A new `NodeError` type in the `yaml` package carries a `*yaml.Node` (line/column) without file path. `UnmarshalYAML` methods in `dto.go` return `NodeError` instead of `fmt.Errorf`. In `loadRecursive`, after `parseRaw` fails, `errors.As` extracts the `NodeError` and converts it to `srcloc.Error` by combining the node with the file path.

**Tech Stack:** Go, `gopkg.in/yaml.v3`, existing `srcloc` package

---

### File Structure

| Action | File | Responsibility |
|--------|------|----------------|
| Create | `yaml/node_error.go` | `NodeError` type, `Error()`, `Unwrap()`, `nodeErrorf()`, `wrapNodeError()` |
| Create | `yaml/node_error_test.go` | Unit tests for `NodeError` |
| Modify | `yaml/dto.go:69-92,120-168,193-214` | Replace `fmt.Errorf` with `nodeErrorf`/`wrapNodeError` at 7 error sites |
| Modify | `yaml/config_loader_yaml.go:63-66` | Add `NodeError` → `srcloc.Error` enrichment after `parseRaw` |
| Modify | `yaml/config_loader_yaml_test.go` | Integration tests: malformed YAML → `srcloc.Error` with correct location |

---

### Task 1: NodeError type

**Files:**
- Create: `yaml/node_error.go`
- Create: `yaml/node_error_test.go`

- [ ] **Step 1: Write failing tests for NodeError**

Create `yaml/node_error_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -run 'TestNodeError|TestWrapNodeError|TestNodeErrorf' ./yaml/ -v`
Expected: compilation error — `NodeError`, `nodeErrorf`, `wrapNodeError` not defined.

- [ ] **Step 3: Implement NodeError**

Create `yaml/node_error.go`:

```go
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
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -run 'TestNodeError|TestWrapNodeError|TestNodeErrorf' ./yaml/ -v`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add yaml/node_error.go yaml/node_error_test.go
git commit -m "Add NodeError type for YAML unmarshal location tracking"
```

---

### Task 2: Convert dto.go error sites to NodeError

**Files:**
- Modify: `yaml/dto.go:69-92` (RawImport)
- Modify: `yaml/dto.go:120-168` (RawServiceTag)
- Modify: `yaml/dto.go:193-214` (RawService)

- [ ] **Step 1: Write failing test for RawImport NodeError**

Add to `yaml/node_error_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -run TestRawImport_UnmarshalYAML_NodeError ./yaml/ -v`
Expected: FAIL — errors.As does not find NodeError (still `fmt.Errorf`).

- [ ] **Step 3: Convert RawImport errors in dto.go**

In `yaml/dto.go`, replace in `RawImport.UnmarshalYAML`:

```go
// Line 85: replace
//   return fmt.Errorf("import path is required")
// with:
return nodeErrorf(node, "import path is required")

// Line 90: replace
//   return fmt.Errorf("import must be a string or mapping")
// with:
return nodeErrorf(node, "import must be a string or mapping")
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -run TestRawImport_UnmarshalYAML_NodeError ./yaml/ -v`
Expected: PASS.

- [ ] **Step 5: Write failing test for RawServiceTag NodeError**

Add to `yaml/node_error_test.go`:

```go
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
```

- [ ] **Step 6: Run test to verify it fails**

Run: `go test -run TestRawServiceTag_UnmarshalYAML_NodeError ./yaml/ -v`
Expected: FAIL.

- [ ] **Step 7: Convert RawServiceTag errors in dto.go**

In `yaml/dto.go`, replace in `RawServiceTag.UnmarshalYAML`:

```go
// Line 128: replace
//   return fmt.Errorf("failed to decode tag name: %w", err)
// with:
return wrapNodeError(node, "failed to decode tag name", err)

// Line 131: replace
//   return fmt.Errorf("tag name is required")
// with:
return nodeErrorf(node, "tag name is required")

// Line 138: replace
//   return fmt.Errorf("tag must be a string or mapping")
// with:
return nodeErrorf(node, "tag must be a string or mapping")

// Line 151: replace
//   return fmt.Errorf("failed to decode tag name: %w", err)
// with:
return wrapNodeError(node, "failed to decode tag name", err)

// Line 157: replace
//   return fmt.Errorf("failed to decode tag attribute %q: %w", key, err)
// with:
return wrapNodeError(valueNode, fmt.Sprintf("failed to decode tag attribute %q", key), err)

// Line 164: replace
//   return fmt.Errorf("tag name is required")
// with:
return nodeErrorf(node, "tag name is required")
```

- [ ] **Step 8: Run test to verify it passes**

Run: `go test -run TestRawServiceTag_UnmarshalYAML_NodeError ./yaml/ -v`
Expected: PASS.

- [ ] **Step 9: Write failing test for RawService NodeError**

Add to `yaml/node_error_test.go`:

```go
func TestRawService_UnmarshalYAML_NodeError(t *testing.T) {
	yaml_input := "[1, 2]"
	var node yaml.Node
	if err := yaml.Unmarshal([]byte(yaml_input), &node); err != nil {
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
```

- [ ] **Step 10: Run test to verify it fails**

Run: `go test -run TestRawService_UnmarshalYAML_NodeError ./yaml/ -v`
Expected: FAIL.

- [ ] **Step 11: Convert RawService error in dto.go**

In `yaml/dto.go`, replace in `RawService.UnmarshalYAML`:

```go
// Line 212: replace
//   return fmt.Errorf("service must be a mapping or alias")
// with:
return nodeErrorf(node, "service must be a mapping or alias")
```

- [ ] **Step 12: Run all tests to verify nothing is broken**

Run: `go test ./yaml/ -v`
Expected: all PASS.

- [ ] **Step 13: Commit**

```bash
git add yaml/dto.go yaml/node_error_test.go
git commit -m "Use NodeError in UnmarshalYAML methods"
```

---

### Task 3: Enrichment in loadRecursive

**Files:**
- Modify: `yaml/config_loader_yaml.go:63-66`
- Modify: `yaml/config_loader_yaml_test.go`

- [ ] **Step 1: Write failing integration test**

Add to `yaml/config_loader_yaml_test.go`:

```go
func TestLoad_UnmarshalError_HasLocation(t *testing.T) {
	tests := []struct {
		name       string
		yaml       string
		wantLine   int
		wantColumn int
		wantMsg    string
	}{
		{
			name: "import missing path",
			yaml: "imports:\n  - exclude:\n      - foo",
			// The mapping node for the import starts at line 2
			wantLine: 2,
			wantMsg:  "import path is required",
		},
		{
			name: "service wrong type",
			yaml: "services:\n  my_svc:\n    - item",
			// The sequence node starts at line 3
			wantLine: 3,
			wantMsg:  "service must be a mapping or alias",
		},
		{
			name:    "tag missing name",
			yaml:    "services:\n  my_svc:\n    constructor:\n      func: fmt.Println\n    tags:\n      - \"\"",
			wantLine: 6,
			wantMsg: "tag name is required",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := writeFile(t, dir, "gendi.yaml", tt.yaml)

			loader := NewConfigLoaderYaml(stubResolver{}, NewParser())
			_, err := loader.Load(path)
			if err == nil {
				t.Fatal("expected error")
			}

			var locErr *srcloc.Error
			if !errors.As(err, &locErr) {
				t.Fatalf("expected srcloc.Error, got %T: %v", err, err)
			}
			if locErr.Loc == nil {
				t.Fatal("expected non-nil location")
			}
			if locErr.Loc.Line != tt.wantLine {
				t.Errorf("Line = %d, want %d", locErr.Loc.Line, tt.wantLine)
			}
			if locErr.Message != tt.wantMsg {
				t.Errorf("Message = %q, want %q", locErr.Message, tt.wantMsg)
			}
			// File should be the absolute path
			if locErr.Loc.File == "" {
				t.Error("expected non-empty File in location")
			}
		})
	}
}
```

This test requires adding imports to `config_loader_yaml_test.go`:

```go
import (
	// ... existing imports ...
	"errors"

	"github.com/asp24/gendi/srcloc"
)
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -run TestLoad_UnmarshalError_HasLocation ./yaml/ -v`
Expected: FAIL — `errors.As` does not find `srcloc.Error` (errors are still plain strings or `NodeError` without enrichment).

- [ ] **Step 3: Add enrichment to loadRecursive**

In `yaml/config_loader_yaml.go`, replace lines 63-66:

```go
// Before:
raw, err := l.parseRaw(data)
if err != nil {
    return nil, fmt.Errorf("parse %s: %w", abs, err)
}

// After:
raw, err := l.parseRaw(data)
if err != nil {
    var ne *NodeError
    if errors.As(err, &ne) {
        loc := srcloc.NewLocation(abs, ne.Node)
        return nil, srcloc.WrapError(loc, ne.Msg, ne.Err)
    }
    return nil, fmt.Errorf("parse %s: %w", abs, err)
}
```

Add imports to `config_loader_yaml.go`:

```go
import (
	// ... existing imports ...
	"errors"

	"github.com/asp24/gendi/srcloc"
)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -run TestLoad_UnmarshalError_HasLocation ./yaml/ -v`
Expected: PASS.

- [ ] **Step 5: Run all tests**

Run: `go test ./... -count=1`
Expected: all PASS.

- [ ] **Step 6: Commit**

```bash
git add yaml/config_loader_yaml.go yaml/config_loader_yaml_test.go
git commit -m "Enrich YAML unmarshal errors with source location"
```
