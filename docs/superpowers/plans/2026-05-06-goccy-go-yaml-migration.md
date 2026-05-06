# goccy/go-yaml Migration — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace `gopkg.in/yaml.v3` with `github.com/goccy/go-yaml v1.19.2` so YAML parse errors render through `srcloc.Renderer` with file:line:col + source snippet, the same way IR validation errors already do.

**Architecture:** Single-shot library swap inside the `yaml/` package. `srcloc/` becomes YAML-library-neutral. New helper `srcloc.AddContext` folds prefix context into existing `*srcloc.Error` without double-locating. New `toSrclocError` at the loader boundary converts goccy's `yaml.Error` interface to `*srcloc.Error`. The CLI wrapper for `load config` is migrated to `AddContext`; `apply passes` and `generate` wrappers are explicitly out of scope per the spec.

**Tech Stack:** Go 1.25.4, `github.com/goccy/go-yaml v1.19.2` (replaces `gopkg.in/yaml.v3`), existing `srcloc.Renderer`, `go test ./...`, `just gen-examples`.

**Spec:** `docs/superpowers/specs/2026-05-06-goccy-go-yaml-migration-design.md`

---

## File Structure

**New files:**
- `srcloc/addcontext.go` — `AddContext` helper.
- `srcloc/addcontext_test.go` — unit tests for `AddContext`.
- `yaml/locations.go` — `newLocation(filePath, ast.Node) *srcloc.Location` helper.
- `yaml/syntax_error_test.go` — new tests for goccy parse-error → `*srcloc.Error` conversion, block scalars, end-to-end load-config rendering.

**Modified files:**
- `go.mod`, `go.sum` — pin `github.com/goccy/go-yaml v1.19.2`, remove `gopkg.in/yaml.v3`.
- `srcloc/location.go` — `NewLocation` signature `(string, *yaml.Node) → (string, int, int)`. Drop `gopkg.in/yaml.v3` import.
- `srcloc/location_test.go` — adapt call sites.
- `yaml/dto.go` — every `UnmarshalYAML(*yaml.Node)` → `UnmarshalYAML(ast.Node)`. `Raw*.Node *yaml.Node` → `Raw*.Node ast.Node`.
- `yaml/parser.go` — `convertLiteral(node *yaml.Node) → convertLiteral(node ast.Node, filePath string)`. Replace `fmt.Errorf("...: %w", err)` wrappers (lines 45, 82, 93, 193) with `srcloc.AddContext`. Substitute `srcloc.NewLocation(filePath, raw.Node)` → `newLocation(filePath, raw.Node)`.
- `yaml/config_loader_yaml.go` — `parseRaw` calls goccy `yaml.Unmarshal`. New `toSrclocError` method. Replace `fmt.Errorf("convert %s: %w", ...)` and the inline parse-error path with `srcloc.AddContext` / `toSrclocError`.
- `yaml/node_error.go` — `NodeError.Node *yaml.Node` → `NodeError.Node ast.Node`. `nodeErrorf(*yaml.Node, ...)` / `wrapNodeError(*yaml.Node, ...)` adapt accordingly.
- `yaml/parser_test.go`, `yaml/node_error_test.go`, `yaml/config_loader_yaml_test.go` — adapt every place that builds a `*yaml.Node` literal or calls `yaml.Unmarshal` to use goccy AST equivalents.
- `cmd/cli.go:34` — `fmt.Errorf("load config: %w", err)` → `srcloc.AddContext(err, "load config")`.

**Untouched (per spec):** `config.go`, all `ir/*`, all `generator/*`, `pipeline/*`, `cmd/cli.go:29/39/44/49`, `config_loader_yaml.go:55/81/86/163`.

---

## Task 1: Add `srcloc.AddContext` helper

**Files:**
- Create: `srcloc/addcontext.go`
- Test: `srcloc/addcontext_test.go`

- [ ] **Step 1.1: Write failing test for nil input**

Create `srcloc/addcontext_test.go`:

```go
package srcloc

import (
	"errors"
	"fmt"
	"testing"
)

func TestAddContext_NilInput(t *testing.T) {
	if got := AddContext(nil, "anything"); got != nil {
		t.Errorf("AddContext(nil, ...) = %v, want nil", got)
	}
}

func TestAddContext_DirectLocatedError_FoldsPrefix(t *testing.T) {
	loc := &Location{File: "/a/b.yaml", Line: 5, Column: 3}
	original := &Error{Loc: loc, Message: "bad value"}

	got := AddContext(original, "parameter %q", "foo")

	want := `/a/b.yaml:5:3: parameter "foo": bad value`
	if got.Error() != want {
		t.Errorf("Error() = %q, want %q", got.Error(), want)
	}
	var le *Error
	if !errors.As(got, &le) {
		t.Fatal("expected *Error in chain")
	}
	if le.Loc != loc {
		t.Errorf("Loc not preserved")
	}
}

func TestAddContext_PlainError_FallsBack(t *testing.T) {
	original := errors.New("boom")

	got := AddContext(original, "context")

	if got.Error() != "context: boom" {
		t.Errorf("Error() = %q, want %q", got.Error(), "context: boom")
	}
	if !errors.Is(got, original) {
		t.Error("errors.Is should find original")
	}
}

func TestAddContext_WrappedLocatedError_DoesNotFoldOuter(t *testing.T) {
	loc := &Location{File: "/a.yaml", Line: 1, Column: 1}
	inner := &Error{Loc: loc, Message: "inner"}
	wrapped := fmt.Errorf("outer: %w", inner)

	got := AddContext(wrapped, "prefix")

	// "outer" must be preserved — not silently discarded.
	if got.Error() != "prefix: outer: /a.yaml:1:1: inner" {
		t.Errorf("Error() = %q", got.Error())
	}
	// Inner *Error still findable by errors.As (so Renderer can find Loc).
	var le *Error
	if !errors.As(got, &le) {
		t.Fatal("expected *Error reachable via errors.As")
	}
	if le.Loc != loc {
		t.Errorf("expected inner Loc, got %v", le.Loc)
	}
}

func TestAddContext_PreservesInnerErr(t *testing.T) {
	cause := errors.New("cause")
	original := &Error{Loc: nil, Message: "wrapped", Err: cause}

	got := AddContext(original, "ctx")

	if !errors.Is(got, cause) {
		t.Error("errors.Is should still find cause")
	}
	if got.Error() != "ctx: wrapped: cause" {
		t.Errorf("Error() = %q", got.Error())
	}
}
```

- [ ] **Step 1.2: Run tests to verify they fail with build error**

Run: `go test ./srcloc/ -run TestAddContext -v`

Expected: build failure — `undefined: AddContext`.

- [ ] **Step 1.3: Implement `AddContext`**

Create `srcloc/addcontext.go`:

```go
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
```

- [ ] **Step 1.4: Run tests to verify they pass**

Run: `go test ./srcloc/ -run TestAddContext -v`

Expected: PASS for all five tests.

- [ ] **Step 1.5: Commit**

```bash
git add srcloc/addcontext.go srcloc/addcontext_test.go
git commit -m "Add srcloc.AddContext for prefix-folding on located errors"
```

---

## Task 2: Pin `github.com/goccy/go-yaml v1.19.2`

**Files:**
- Modify: `go.mod`, `go.sum`

- [ ] **Step 2.1: Add the dependency**

Run: `go get github.com/goccy/go-yaml@v1.19.2`

Expected: `go.mod` gets new `require` line, `go.sum` updates. `gopkg.in/yaml.v3` still present.

- [ ] **Step 2.2: Verify nothing broke**

Run: `go build ./...` and `go test ./...`

Expected: both PASS — we only added the dep, no code uses it yet.

- [ ] **Step 2.3: Commit**

```bash
git add go.mod go.sum
git commit -m "Add github.com/goccy/go-yaml v1.19.2 dependency"
```

---

## Task 3: Write failing syntax-error tests

These tests describe the post-migration behavior. They will fail until Task 5 ships, that is the point.

**Files:**
- Create: `yaml/syntax_error_test.go`

- [ ] **Step 3.1: Write the test file**

Create `yaml/syntax_error_test.go`:

```go
package yaml

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/asp24/gendi/srcloc"
)

func writeYAML(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "gendi.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp yaml: %v", err)
	}
	return path
}

func TestSyntaxError_BadIndent_ProducesSrclocError(t *testing.T) {
	// Indentation under "services" is broken: "foo" and "baz" disagree.
	yaml := "services:\n foo: bar\n  baz: qux\n"
	path := writeYAML(t, yaml)

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var le *srcloc.Error
	if !errors.As(err, &le) {
		t.Fatalf("expected *srcloc.Error, got %T: %v", err, err)
	}
	if le.Loc == nil {
		t.Fatal("expected non-nil Loc")
	}
	absPath, _ := filepath.Abs(path)
	if le.Loc.File != absPath {
		t.Errorf("Loc.File = %q, want %q", le.Loc.File, absPath)
	}
	if le.Loc.Line < 1 {
		t.Errorf("Loc.Line = %d, want >= 1", le.Loc.Line)
	}
}

// Spec §Testing #1: exact line/column assertion (catches off-by-one
// regressions when goccy version changes). The exact values must be
// confirmed by running this test once after Task 5 lands and updating
// wantLine/wantCol if the initial implementer's expectation differs by
// 1 — pick the values goccy reports for v1.19.2 and freeze them.
func TestSyntaxError_BadIndent_ExactLocation(t *testing.T) {
	// Carefully-controlled YAML: the second mapping key under "services"
	// has one extra space of indent, error is reported at that key.
	//
	// Line 1: services:
	// Line 2:  foo: bar       (1-space indent)
	// Line 3:   baz: qux      (2-space indent — the conflict)
	yaml := "services:\n foo: bar\n  baz: qux\n"
	path := writeYAML(t, yaml)

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error")
	}
	var le *srcloc.Error
	if !errors.As(err, &le) {
		t.Fatalf("expected *srcloc.Error, got %T", err)
	}
	if le.Loc == nil {
		t.Fatal("nil Loc")
	}
	// Replace these with the exact values goccy v1.19.2 reports for the
	// above YAML. After running the test once, lock these in.
	const wantLine, wantCol = 3, 3
	if le.Loc.Line != wantLine || le.Loc.Column != wantCol {
		t.Errorf("Loc = %d:%d, want %d:%d (lock these in once verified for goccy v1.19.2)",
			le.Loc.Line, le.Loc.Column, wantLine, wantCol)
	}
}

func TestSyntaxError_UnclosedQuote_ProducesSrclocError(t *testing.T) {
	yaml := `parameters:
  bad:
    type: string
    value: "unclosed
`
	path := writeYAML(t, yaml)

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error")
	}
	var le *srcloc.Error
	if !errors.As(err, &le) {
		t.Fatalf("expected *srcloc.Error, got %T: %v", err, err)
	}
	if le.Loc == nil {
		t.Fatal("expected non-nil Loc")
	}
}

func TestSyntaxError_TypeMismatch_ProducesSrclocError(t *testing.T) {
	// decoration_priority is int, not string. Triggers a yaml.Error
	// that is NOT a *yaml.SyntaxError (decode error path).
	yaml := `services:
  foo:
    type: string
    constructor:
      func: pkg.New
    decoration_priority: "abc"
`
	path := writeYAML(t, yaml)

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error")
	}
	var le *srcloc.Error
	if !errors.As(err, &le) {
		t.Fatalf("expected *srcloc.Error, got %T: %v", err, err)
	}
}

func TestSyntaxError_RenderingProducesSnippet(t *testing.T) {
	// End-to-end: feed a known-bad YAML, render the error, expect a
	// caret-style snippet output.
	yaml := "services:\n foo: bar\n  baz: qux\n"
	path := writeYAML(t, yaml)

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error")
	}
	rendered := srcloc.NewRenderer().RenderError(err, 2)
	if !strings.Contains(rendered, "^") {
		t.Errorf("expected caret in rendered output, got:\n%s", rendered)
	}
	if !strings.Contains(rendered, "gendi.yaml:") {
		t.Errorf("expected file:line:col prefix, got:\n%s", rendered)
	}
}

func TestSyntaxError_EmptyFile_DoesNotPanic(t *testing.T) {
	path := writeYAML(t, "")
	// Empty file should either return nil config (if goccy treats it as
	// nil doc) or a srcloc.Error without panic. The point is: no panic.
	_, _ = LoadConfig(path)
}
```

- [ ] **Step 3.2: Run tests to confirm they fail**

Run: `go test ./yaml/ -run TestSyntaxError -v`

Expected: most tests FAIL (errors come back wrapped as `parse /path: yaml: line N: ...` or as `*fmt.wrapError`, not as `*srcloc.Error`). `TestSyntaxError_EmptyFile_DoesNotPanic` may pass (no panic in the current impl).

This is the acceptance baseline for Task 5.

- [ ] **Step 3.3: Commit**

```bash
git add yaml/syntax_error_test.go
git commit -m "Add failing tests for goccy-style parse-error rendering"
```

(Note: tests are committed in the failing state intentionally; they pass after Task 5.)

---

## Task 4: `srcloc.NewLocation` becomes YAML-library-neutral

**Files:**
- Modify: `srcloc/location.go`
- Modify: `srcloc/location_test.go`
- Modify: `yaml/parser.go` — adapt call sites to pass `int, int`
- Modify: `yaml/config_loader_yaml.go` — adapt call sites
- Modify: `yaml/node_error_test.go` (no — this stays on yaml.v3 for now; test only adapts in Task 5)

- [ ] **Step 4.1: Modify `srcloc/location.go`**

Replace the file contents with:

```go
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
```

- [ ] **Step 4.2: Adapt `srcloc/location_test.go`**

Replace with:

```go
package srcloc

import "testing"

func TestNewLocation(t *testing.T) {
	loc := NewLocation("/a/b.yaml", 5, 3)
	if loc == nil {
		t.Fatal("expected non-nil")
	}
	if loc.File != "/a/b.yaml" || loc.Line != 5 || loc.Column != 3 {
		t.Errorf("unexpected location: %+v", loc)
	}
}

func TestNewLocation_EmptyFile_ReturnsNil(t *testing.T) {
	if NewLocation("", 5, 3) != nil {
		t.Error("expected nil for empty file")
	}
}

func TestNewLocation_ZeroLine_ReturnsNil(t *testing.T) {
	if NewLocation("/a.yaml", 0, 1) != nil {
		t.Error("expected nil for zero line")
	}
}

func TestLocation_String(t *testing.T) {
	loc := NewLocation("/a.yaml", 5, 3)
	if got := loc.String(); got != "/a.yaml:5:3" {
		t.Errorf("String() = %q", got)
	}
	var nilLoc *Location
	if got := nilLoc.String(); got != "" {
		t.Errorf("nil String() = %q", got)
	}
}
```

- [ ] **Step 4.3: Adapt all call sites in `yaml/parser.go`**

Find every `srcloc.NewLocation(filePath, X.Node)` and rewrite as `srcloc.NewLocation(filePath, X.Node.Line, X.Node.Column)`. There are six call sites (lines 51, 68, 141, 158, 166, 218 by the spec's numbering). For each, also add a nil-guard if `X.Node` may be nil — current code dereferences blindly, so keep the same semantics for now (TestSet stays the same).

Concrete edits in `yaml/parser.go`:

- line 51: `SourceLoc: srcloc.NewLocation(filePath, param.Node)` → `SourceLoc: nodeLoc(filePath, param.Node)` — but `nodeLoc` does not exist yet at this stage. Use the inline form for now:
  - `SourceLoc: srcloc.NewLocation(filePath, param.Node.Line, param.Node.Column)` — guarded by an `if param.Node != nil` block, falling back to `nil` Loc when nil.

Actually, simpler: add a tiny local helper inline in parser.go for now, replaced in Task 5 by `newLocation(filePath, ast.Node)`. To avoid that churn, just inline:

```go
// before
SourceLoc: srcloc.NewLocation(filePath, param.Node),

// after
SourceLoc: locFromYamlNode(filePath, param.Node),
```

Add at the bottom of `yaml/parser.go`:

```go
import "gopkg.in/yaml.v3"

// locFromYamlNode is a temporary adapter; replaced by newLocation in
// Task 5 once goccy migration is complete.
func locFromYamlNode(filePath string, n *yaml.Node) *srcloc.Location {
	if n == nil {
		return nil
	}
	return srcloc.NewLocation(filePath, n.Line, n.Column)
}
```

Replace all six `srcloc.NewLocation(filePath, …Node)` calls with `locFromYamlNode(filePath, …Node)`.

- [ ] **Step 4.4: Adapt the call site in `yaml/config_loader_yaml.go`**

In the `errors.As(err, &ne)` branch (around line 69), replace:

```go
loc := srcloc.NewLocation(abs, ne.Node)
return nil, srcloc.WrapError(loc, ne.Msg, ne.Err)
```

with:

```go
loc := locFromYamlNode(abs, ne.Node)
return nil, srcloc.WrapError(loc, ne.Msg, ne.Err)
```

(`locFromYamlNode` is exported only to package, defined in parser.go.)

- [ ] **Step 4.5: Run all tests**

Run: `go test ./...`

Expected: all existing tests PASS. `TestSyntaxError_*` from Task 3 still FAIL (we have not migrated parser yet).

- [ ] **Step 4.6: Commit**

```bash
git add srcloc/location.go srcloc/location_test.go yaml/parser.go yaml/config_loader_yaml.go
git commit -m "Make srcloc.NewLocation YAML-library-neutral"
```

---

## Task 5: Atomic migration of `yaml/*` to goccy

This is the biggest task. The yaml package switches from `gopkg.in/yaml.v3` to `github.com/goccy/go-yaml` in a single coherent change. After this task the package compiles, all old tests pass (after mechanical adaptation), and the Task 3 syntax-error tests pass.

**Files:**
- Modify: `yaml/dto.go`
- Modify: `yaml/parser.go`
- Modify: `yaml/config_loader_yaml.go`
- Modify: `yaml/node_error.go`
- Modify: `yaml/parser_test.go`, `yaml/node_error_test.go`, `yaml/config_loader_yaml_test.go`
- Create: `yaml/locations.go`

- [ ] **Step 5.1: Create `yaml/locations.go`**

```go
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
```

- [ ] **Step 5.2: Rewrite `yaml/node_error.go`**

```go
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
```

- [ ] **Step 5.3: Rewrite `yaml/dto.go`**

Replace the entire file with the goccy version. Below is the full target. Read the existing `yaml/dto.go` to make sure no field is dropped before pasting.

**API confirmations (verified in `~/go/pkg/mod/github.com/goccy/go-yaml@v1.19.2`):**
- `yaml.NodeUnmarshaler { UnmarshalYAML(ast.Node) error }` exists (`yaml.go:61`).
- `yaml.NodeToValue(node, &v) error` exists (`yaml.go:217`).
- Struct fields of type `ast.Node` are special-cased — the decoder sets them directly to the raw AST node (`decode.go:939`). So `RawParameter.Value ast.Node` works without a custom `UnmarshalYAML`.
- `*ast.MappingNode { Values []*MappingValueNode }`; `*ast.MappingValueNode { Key MapKeyNode; Value ast.Node }`. `MapKeyNode` embeds `ast.Node`, so passing `kv.Key` to a function expecting `ast.Node` compiles.
- AST node types referenced: `*ast.StringNode { Value string }`, `*ast.LiteralNode { Value *StringNode }`, `*ast.IntegerNode { Value interface{} (int64 or uint64) }`, `*ast.FloatNode { Value float64 }`, `*ast.BoolNode { Value bool }`, `*ast.NullNode`, `*ast.InfinityNode`, `*ast.NanNode` — all present.
- `yaml.Error` is a public alias of `errors.Error` interface with `error`, `GetToken() *token.Token`, `GetMessage() string`. Implementations include `*SyntaxError`, `*TypeError`, `*OverflowError`, `*DuplicateKeyError`, `*UnknownFieldError`, `*UnexpectedNodeTypeError`.

```go
package yaml

import (
	"fmt"

	yamllib "github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
)

// RawConfig is the YAML-specific representation of a config file.
type RawConfig struct {
	Imports    []RawImport             `yaml:"imports"`
	Parameters map[string]RawParameter `yaml:"parameters"`
	Tags       map[string]RawTag       `yaml:"tags"`
	Services   map[string]*RawService  `yaml:"services"`
}

func (c *RawConfig) UnmarshalYAML(node ast.Node) error {
	type alias RawConfig
	var decoded alias
	if err := yamllib.NodeToValue(node, &decoded); err != nil {
		return err
	}

	mapping, ok := node.(*ast.MappingNode)
	if ok {
		for _, kv := range mapping.Values {
			keyStr := keyString(kv.Key)
			switch keyStr {
			case "parameters":
				if pm, ok := kv.Value.(*ast.MappingNode); ok {
					for _, pv := range pm.Values {
						name := keyString(pv.Key)
						if param, ok := decoded.Parameters[name]; ok {
							param.Node = pv.Value
							decoded.Parameters[name] = param
						}
					}
				}
			case "tags":
				if tm, ok := kv.Value.(*ast.MappingNode); ok {
					for _, tv := range tm.Values {
						name := keyString(tv.Key)
						if tag, ok := decoded.Tags[name]; ok {
							tag.Node = tv.Value
							decoded.Tags[name] = tag
						}
					}
				}
			}
		}
	}

	*c = RawConfig(decoded)
	return nil
}

// keyString extracts a scalar string from a key node. Returns "" for
// any non-scalar key (which never occurs in well-formed configs).
func keyString(n ast.Node) string {
	if s, ok := n.(*ast.StringNode); ok {
		return s.Value
	}
	return ""
}

type RawImport struct {
	Path    string   `yaml:"path"`
	Exclude []string `yaml:"exclude"`
}

func (i *RawImport) UnmarshalYAML(node ast.Node) error {
	switch n := node.(type) {
	case *ast.StringNode:
		i.Path = n.Value
		return nil
	case *ast.LiteralNode:
		if n.Value != nil {
			i.Path = n.Value.Value
		}
		return nil
	case *ast.MappingNode:
		type alias RawImport
		var decoded alias
		if err := yamllib.NodeToValue(node, &decoded); err != nil {
			return err
		}
		if decoded.Path == "" {
			return nodeErrorf(node, "import path is required")
		}
		*i = RawImport(decoded)
		return nil
	default:
		return nodeErrorf(node, "import must be a string or mapping")
	}
}

type RawParameter struct {
	Type  string   `yaml:"type"`
	Value ast.Node `yaml:"value"`

	// Node holds the full parameter mapping node for location tracking.
	Node ast.Node `yaml:"-"`
}

type RawTag struct {
	ElementType   string `yaml:"element_type"`
	SortBy        string `yaml:"sort_by"`
	Public        bool   `yaml:"public"`
	Autoconfigure bool   `yaml:"autoconfigure"`

	// Node holds the tag mapping node for location tracking.
	Node ast.Node `yaml:"-"`
}

type RawServiceTag struct {
	Name       string
	Attributes map[string]interface{}

	// Node holds the tag mapping node for location tracking.
	Node ast.Node `yaml:"-"`
}

func (t *RawServiceTag) UnmarshalYAML(node ast.Node) error {
	t.Node = node
	t.Attributes = make(map[string]interface{})

	switch n := node.(type) {
	case *ast.StringNode:
		if n.Value == "" {
			return nodeErrorf(node, "tag name is required")
		}
		t.Name = n.Value
		return nil
	case *ast.LiteralNode:
		if n.Value == nil || n.Value.Value == "" {
			return nodeErrorf(node, "tag name is required")
		}
		t.Name = n.Value.Value
		return nil
	case *ast.MappingNode:
		for _, kv := range n.Values {
			key := keyString(kv.Key)
			if key == "name" {
				if err := yamllib.NodeToValue(kv.Value, &t.Name); err != nil {
					return wrapNodeError(node, "failed to decode tag name", err)
				}
			} else {
				var value interface{}
				if err := yamllib.NodeToValue(kv.Value, &value); err != nil {
					return wrapNodeError(kv.Value, fmt.Sprintf("failed to decode tag attribute %q", key), err)
				}
				t.Attributes[key] = value
			}
		}
		if t.Name == "" {
			return nodeErrorf(node, "tag name is required")
		}
		return nil
	default:
		return nodeErrorf(node, "tag must be a string or mapping")
	}
}

// ServiceDefaults holds default values for service configuration.
type ServiceDefaults struct {
	Shared        *bool `yaml:"shared"`
	Public        *bool `yaml:"public"`
	Autoconfigure *bool `yaml:"autoconfigure"`
}

type RawService struct {
	Type               string          `yaml:"type"`
	Constructor        RawConstructor  `yaml:"constructor"`
	Shared             *bool           `yaml:"shared"`
	Public             *bool           `yaml:"public"`
	Autoconfigure      *bool           `yaml:"autoconfigure"`
	Decorates          string          `yaml:"decorates"`
	DecorationPriority int             `yaml:"decoration_priority"`
	Tags               []RawServiceTag `yaml:"tags"`
	Alias              string          `yaml:"alias"`

	// Node holds the service mapping node for location tracking.
	Node ast.Node `yaml:"-"`
}

func (s *RawService) UnmarshalYAML(node ast.Node) error {
	switch n := node.(type) {
	case *ast.StringNode:
		*s = RawService{Alias: n.Value, Node: node}
		return nil
	case *ast.LiteralNode:
		ref := ""
		if n.Value != nil {
			ref = n.Value.Value
		}
		*s = RawService{Alias: ref, Node: node}
		return nil
	case *ast.MappingNode:
		type alias RawService
		var decoded alias
		if err := yamllib.NodeToValue(node, &decoded); err != nil {
			return err
		}
		*s = RawService(decoded)
		s.Node = node
		return nil
	default:
		return nodeErrorf(node, "service must be a mapping or alias")
	}
}

type RawConstructor struct {
	Func   string        `yaml:"func"`
	Method string        `yaml:"method"`
	Args   []RawArgument `yaml:"args"`

	// Node points at func/method scalar for location tracking.
	Node ast.Node `yaml:"-"`
}

func (c *RawConstructor) UnmarshalYAML(node ast.Node) error {
	c.Node = node

	type raw struct {
		Func   string     `yaml:"func"`
		Method string     `yaml:"method"`
		Args   []ast.Node `yaml:"args"`
	}
	var decoded raw
	if err := yamllib.NodeToValue(node, &decoded); err != nil {
		return err
	}
	c.Func = decoded.Func
	c.Method = decoded.Method

	if mapping, ok := node.(*ast.MappingNode); ok {
		for _, kv := range mapping.Values {
			switch keyString(kv.Key) {
			case "func", "method":
				c.Node = kv.Value
			}
		}
	}

	if len(decoded.Args) == 0 {
		return nil
	}
	c.Args = make([]RawArgument, len(decoded.Args))
	for i := range decoded.Args {
		if err := c.Args[i].UnmarshalYAML(decoded.Args[i]); err != nil {
			return err
		}
	}
	return nil
}

type RawArgument struct {
	Value *string
	Node  ast.Node
}

func (a *RawArgument) UnmarshalYAML(node ast.Node) error {
	a.Node = node

	switch n := node.(type) {
	case *ast.StringNode:
		val := n.Value
		a.Value = &val
	case *ast.LiteralNode:
		if n.Value != nil {
			val := n.Value.Value
			a.Value = &val
		}
	}
	return nil
}
```

- [ ] **Step 5.4: Rewrite `yaml/parser.go`**

This is the most intricate file. Concrete edits below:

a) Replace the import block:

```go
import (
	"fmt"
	"math"
	"strings"

	yamllib "github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"

	di "github.com/asp24/gendi"
	"github.com/asp24/gendi/srcloc"
	"github.com/asp24/gendi/typeres"
)
```

(`yamllib` is unused inside parser.go after this, but kept for `convertLiteral`'s integer-overflow path if needed — drop if `goimports` complains. `math` is needed for `math.MaxInt64`.)

b) Replace `convertLiteral`:

```go
func (p *Parser) convertLiteral(node ast.Node, filePath string) (di.Literal, error) {
	loc := newLocation(filePath, node)

	switch n := node.(type) {
	case *ast.StringNode:
		return di.NewStringLiteral(n.Value), nil
	case *ast.LiteralNode:
		if n.Value == nil {
			return di.NewStringLiteral(""), nil
		}
		return di.NewStringLiteral(n.Value.Value), nil
	case *ast.IntegerNode:
		switch v := n.Value.(type) {
		case int64:
			return di.NewIntLiteral(v), nil
		case uint64:
			if v > math.MaxInt64 {
				return di.Literal{}, srcloc.Errorf(loc, "integer value %d does not fit in int64", v)
			}
			return di.NewIntLiteral(int64(v)), nil
		default:
			return di.Literal{}, srcloc.Errorf(loc, "unsupported integer kind %T", v)
		}
	case *ast.FloatNode:
		return di.NewFloatLiteral(n.Value), nil
	case *ast.InfinityNode:
		return di.Literal{}, srcloc.Errorf(loc, ".inf is not supported as a literal value")
	case *ast.NanNode:
		return di.Literal{}, srcloc.Errorf(loc, ".nan is not supported as a literal value")
	case *ast.BoolNode:
		return di.NewBoolLiteral(n.Value), nil
	case *ast.NullNode:
		return di.NewNullLiteral(), nil
	default:
		return di.Literal{}, srcloc.Errorf(loc, "unsupported literal type %T", node)
	}
}
```

c) Update every `srcloc.NewLocation(filePath, X.Node)` → `newLocation(filePath, X.Node)`. Six call sites. Drop the `locFromYamlNode` helper added in Task 4.3 (no longer needed; all callers pass goccy `ast.Node`).

d) Replace prefix wrappers (per spec §3.2 Parser):

- `parser.go:41` (terminal): `return nil, fmt.Errorf("parameter %q: type is required", name)` becomes
  `return nil, srcloc.Errorf(newLocation(filePath, param.Node), "parameter %q: type is required", name)`.
- `parser.go:45`: `return nil, fmt.Errorf("parameter %q: %w", name, err)` becomes
  `return nil, srcloc.AddContext(err, "parameter %q", name)`.
- `parser.go:82`: `return nil, fmt.Errorf("_default: %w", err)` becomes
  `return nil, srcloc.AddContext(err, "_default")`.
- `parser.go:93`: `return nil, fmt.Errorf("service %q: %w", name, err)` becomes
  `return nil, srcloc.AddContext(err, "service %q", name)`.
- `parser.go:193` (inside `convertServiceWithPackageAndFile`): `return di.Service{}, fmt.Errorf("arg[%d]: %w", i, err)` becomes
  `return di.Service{}, srcloc.AddContext(err, "arg[%d]", i)`.
- `parser.go:248`: `return di.Argument{}, fmt.Errorf("argument must have a value")` becomes
  `return di.Argument{}, srcloc.Errorf(newLocation(filePath, raw.Node), "argument must have a value")`.

  (Note: `convertArgumentWithFile` already has `filePath` and `raw.Node` in scope.)

e) Replace the call site `p.convertLiteral(&param.Value)` (line 43) with `p.convertLiteral(param.Value, filePath)` — note `param.Value` is now an `ast.Node`, not `yaml.Node` value. Same for the call inside `convertArgumentWithFile` (line 237): `p.convertLiteral(raw.Node, filePath)`.

f) Update validateDefaults messages (lines 283-298) — these are still `fmt.Errorf(...)` because the wrapper at line 82 now folds them into a `_default` context. They are not located today; leave that for a follow-up.

- [ ] **Step 5.5: Rewrite `yaml/config_loader_yaml.go`**

a) Update import block:

```go
import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	yamllib "github.com/goccy/go-yaml"

	di "github.com/asp24/gendi"
	"github.com/asp24/gendi/imprt"
	"github.com/asp24/gendi/srcloc"
)
```

(Drop `ylib "gopkg.in/yaml.v3"`.)

b) Replace the parse-error block in `loadRecursive` (around lines 60-73):

```go
data, err := l.readFile(abs)
if err != nil {
	return nil, err
}

raw, err := l.parseRaw(data)
if err != nil {
	return nil, l.toSrclocError(abs, err)
}
```

c) Replace `defaultYamlUnmarshal` and `yamlUnmarshal` with the goccy equivalent:

```go
// yamlUnmarshal wraps yaml.Unmarshal for testability.
var defaultYamlUnmarshal = yamllib.Unmarshal

func (l *ConfigLoaderYaml) yamlUnmarshal(data []byte, v interface{}) error {
	return defaultYamlUnmarshal(data, v)
}
```

d) Replace the convert wrapper at line 100:

```go
cfg, err := l.parser.ConvertConfigWithDirAndFile(raw, baseDir, abs)
if err != nil {
	return nil, srcloc.AddContext(err, "convert %s", abs)
}
```

e) Add `toSrclocError` at the bottom of the file:

```go
func (l *ConfigLoaderYaml) toSrclocError(file string, err error) error {
	var ne *NodeError
	if errors.As(err, &ne) {
		loc := newLocation(file, ne.Node)

		// ne.Err may itself wrap a goccy yaml.Error (e.g. NodeToValue
		// failed inside a custom UnmarshalYAML). srcloc.WrapError would
		// call ne.Err.Error(), which routes through goccy's own
		// formatter and breaks the single-style invariant. Normalize:
		// strip the wrapped yaml.Error down to its plain message.
		wrapped := ne.Err
		var inner yamllib.Error
		if errors.As(wrapped, &inner) {
			wrapped = errors.New(inner.GetMessage())
		}
		return srcloc.WrapError(loc, ne.Msg, wrapped)
	}

	var ye yamllib.Error
	if errors.As(err, &ye) {
		var loc *srcloc.Location
		if tok := ye.GetToken(); tok != nil && tok.Position != nil {
			loc = srcloc.NewLocation(file, tok.Position.Line, tok.Position.Column)
		}
		return srcloc.Errorf(loc, "%s", ye.GetMessage())
	}

	return srcloc.AddContext(err, "parse %s", file)
}
```

- [ ] **Step 5.6: Adapt `yaml/parser_test.go`**

Two patterns to fix:

**Pattern A** — manually-built `yaml.Node` literals. Replace each one with a small helper that parses a YAML snippet via goccy and returns the document root, e.g.:

```go
import (
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
)

func mustParseNode(t *testing.T, src string) ast.Node {
	t.Helper()
	f, err := parser.ParseBytes([]byte(src), 0)
	if err != nil {
		t.Fatalf("parse helper: %v", err)
	}
	if len(f.Docs) == 0 {
		t.Fatal("no docs")
	}
	return f.Docs[0].Body
}
```

Add this helper near the top of `parser_test.go` (or in a new `yaml/testhelper_test.go`).

Then rewrite `yaml.Node` literals:

- `yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "foo"}` → `mustParseNode(t, "foo")` — yields a `*ast.StringNode`.
- `yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: "5"}` → `mustParseNode(t, "5")`.
- `yaml.Node{Kind: yaml.ScalarNode, Tag: "!!binary", Value: "x"}` (used to test unsupported tag) → can no longer be constructed because goccy infers types; use `mustParseNode(t, "!!binary x")` and adjust the expected error message accordingly. If the test cannot be cleanly translated, replace with a different unsupported-type case (e.g. a mapping where a literal is expected: `mustParseNode(t, "{a: b}")`).

**Pattern B** — `yaml.Unmarshal([]byte(yamlContent), &raw)` (lines 315, 362) → `yamllib.Unmarshal([]byte(yamlContent), &raw)` where `yamllib` is `github.com/goccy/go-yaml`. Adjust import.

**Important**: tests asserting specific error text from yaml.v3 must be relaxed to `errors.As(&srcloc.Error)` + non-empty `Message`. Quick grep with `go test ./yaml/ -run TestParam... -v` should reveal any.

- [ ] **Step 5.7: Adapt `yaml/node_error_test.go`**

Same pattern. Replace `&yaml.Node{Line: 5, Column: 3}` with a node parsed via `mustParseNode`. For tests that need exact `Line: 5, Column: 3`, parse a multi-line YAML where the target node sits at the right position, e.g.:

```go
// "x: y" on line 5 column 3 — pad with leading newlines and indentation.
node := mustParseNode(t, "\n\n\n\n  x: y").(*ast.MappingNode).Values[0].Value
```

Or, simpler: change the tests to assert "node has a non-nil token with Line >= 1" rather than exact Line: 5. The exact-Line assertions in `node_error_test.go` were artificial test fixtures, not invariants of production code; loosen them.

(If a test specifically validates the `Error()` string format with positions, refactor to assert just the message portion.)

- [ ] **Step 5.8: Adapt `yaml/config_loader_yaml_test.go`**

Verified at planning time: `grep -n 'yaml\.' yaml/config_loader_yaml_test.go` returns zero hits. This file tests file IO and import resolution and never touches yaml internals directly. **No changes required.** Skip this step.

- [ ] **Step 5.9: Run yaml package tests**

Run: `go test ./yaml/... -v`

Expected: all tests PASS — both the migrated old tests AND the four `TestSyntaxError_*` tests added in Task 3.

If a test from Task 3 fails because Loc is nil or wrong type, debug `toSrclocError` first (most likely the goccy error type is not implementing `yaml.Error`).

- [ ] **Step 5.10: Run full test suite**

Run: `go test ./...`

Expected: all PASS.

- [ ] **Step 5.11: Commit**

```bash
git add yaml/dto.go yaml/parser.go yaml/config_loader_yaml.go yaml/node_error.go yaml/locations.go yaml/parser_test.go yaml/node_error_test.go yaml/config_loader_yaml_test.go
git commit -m "Migrate yaml package to github.com/goccy/go-yaml"
```

---

## Task 6: Add `convertLiteral` error-path tests

These were promised by spec §Testing #5 but cannot be added until convertLiteral has the new signature. Now they can.

**Files:**
- Modify: `yaml/parser_test.go`

- [ ] **Step 6.1: Add tests**

Append to `yaml/parser_test.go`:

```go
import "github.com/asp24/gendi/srcloc"  // if not already imported

func TestConvertLiteral_IntegerOverflow_Located(t *testing.T) {
	node := mustParseNode(t, "99999999999999999999")
	p := NewParser()
	_, err := p.convertLiteral(node, "/x.yaml")
	if err == nil {
		t.Fatal("expected error")
	}
	var le *srcloc.Error
	if !errors.As(err, &le) || le.Loc == nil {
		t.Fatalf("expected located *srcloc.Error, got %T: %v", err, err)
	}
}

func TestConvertLiteral_Inf_Rejected(t *testing.T) {
	node := mustParseNode(t, ".inf")
	p := NewParser()
	_, err := p.convertLiteral(node, "/x.yaml")
	if err == nil {
		t.Fatal("expected error")
	}
	var le *srcloc.Error
	if !errors.As(err, &le) {
		t.Fatalf("expected *srcloc.Error, got %T", err)
	}
	if !strings.Contains(le.Message, ".inf") {
		t.Errorf("expected .inf in message, got %q", le.Message)
	}
}

func TestConvertLiteral_Nan_Rejected(t *testing.T) {
	node := mustParseNode(t, ".nan")
	p := NewParser()
	_, err := p.convertLiteral(node, "/x.yaml")
	if err == nil {
		t.Fatal("expected error")
	}
	var le *srcloc.Error
	if !errors.As(err, &le) {
		t.Fatalf("expected *srcloc.Error, got %T", err)
	}
}

func TestConvertLiteral_Mapping_Rejected(t *testing.T) {
	node := mustParseNode(t, "{a: b}")
	p := NewParser()
	_, err := p.convertLiteral(node, "/x.yaml")
	if err == nil {
		t.Fatal("expected error")
	}
	var le *srcloc.Error
	if !errors.As(err, &le) {
		t.Fatalf("expected *srcloc.Error, got %T", err)
	}
}
```

(Add the `errors` and `strings` imports if not present.)

- [ ] **Step 6.2: Run tests**

Run: `go test ./yaml/ -run TestConvertLiteral -v`

Expected: PASS.

- [ ] **Step 6.3: Commit**

```bash
git add yaml/parser_test.go
git commit -m "Test convertLiteral overflow, .inf/.nan, and unsupported-node paths"
```

---

## Task 7: Block / folded scalar tests

Spec §Testing #4 — the `|` and `>` cross product.

**Files:**
- Modify: `yaml/syntax_error_test.go` (add cases there to keep new yaml-behavior tests in one place)

- [ ] **Step 7.1: Append tests**

```go
func TestBlockScalar_Param_Pipe(t *testing.T) {
	yaml := `parameters:
  greeting:
    type: string
    value: |
      hello
      world
`
	path := writeYAML(t, yaml)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := cfg.Parameters["greeting"].Value.String()
	want := "hello\nworld\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBlockScalar_Param_Folded(t *testing.T) {
	yaml := `parameters:
  greeting:
    type: string
    value: >
      hello
      world
`
	path := writeYAML(t, yaml)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := cfg.Parameters["greeting"].Value.String()
	want := "hello world\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBlockScalar_Arg_Pipe(t *testing.T) {
	yaml := `services:
  echo:
    type: string
    constructor:
      func: pkg.New
      args:
        - |
          hello
          world
`
	path := writeYAML(t, yaml)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	args := cfg.Services["echo"].Constructor.Args
	if len(args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(args))
	}
	got := args[0].Literal.String()
	want := "hello\nworld\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBlockScalar_Arg_Folded(t *testing.T) {
	yaml := `services:
  echo:
    type: string
    constructor:
      func: pkg.New
      args:
        - >
          hello
          world
`
	path := writeYAML(t, yaml)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	args := cfg.Services["echo"].Constructor.Args
	if len(args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(args))
	}
	got := args[0].Literal.String()
	want := "hello world\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
```

(`di.Literal.String()` is a method on `Literal` defined in `literal.go:21`. Returns the inner string value if `Kind == LiteralString`, otherwise empty string.)

- [ ] **Step 7.2: Run tests**

Run: `go test ./yaml/ -run TestBlockScalar -v`

Expected: PASS for all four.

- [ ] **Step 7.3: Commit**

```bash
git add yaml/syntax_error_test.go
git commit -m "Test block (|) and folded (>) scalars in params and args"
```

---

## Task 8: `cmd/cli.go:34` → `srcloc.AddContext`

**Files:**
- Modify: `cmd/cli.go`

- [ ] **Step 8.1: Edit `cmd/cli.go`**

Change line 34 from:

```go
diCfg, err := yaml.LoadConfig(cfg.ConfigPath)
if err != nil {
	return fmt.Errorf("load config: %w", err)
}
```

to:

```go
diCfg, err := yaml.LoadConfig(cfg.ConfigPath)
if err != nil {
	return srcloc.AddContext(err, "load config")
}
```

(`srcloc` is already imported in `cmd/cli.go`.)

Lines 39 (`apply passes`) and 44 (`generate`) stay as `fmt.Errorf` — see spec §3.2 "CLI wrappers we deliberately leave alone".

- [ ] **Step 8.2: Run tests**

Run: `go test ./...`

Expected: PASS.

- [ ] **Step 8.3: Commit**

```bash
git add cmd/cli.go
git commit -m "Use srcloc.AddContext for the load-config CLI wrapper"
```

---

## Task 9: End-to-end load-config render test

Spec §Testing #8 — verify the full chain doesn't double-locate or drop context.

**Files:**
- Modify: `yaml/syntax_error_test.go`

- [ ] **Step 9.1: Add e2e test**

Append:

```go
func TestE2E_LoadConfigPath_NoDoubleLocation(t *testing.T) {
	// Triggers convertLiteral overflow inside a parameter value.
	// Chain: convertLiteral -> parser:43 (no wrapping for parameter
	// value-decode location since convertLiteral already locates) ->
	// parser:45 AddContext("parameter %q") ->
	// loader:100 AddContext("convert %s") ->
	// (caller would invoke cli:34 AddContext("load config"))
	yaml := `parameters:
  bignum:
    type: int
    value: 99999999999999999999
`
	path := writeYAML(t, yaml)
	absPath, _ := filepath.Abs(path)

	loaderErr := func() error {
		_, err := LoadConfig(path)
		return err
	}()
	if loaderErr == nil {
		t.Fatal("expected loader error")
	}

	// Simulate cli:34 wrapper.
	withCLI := srcloc.AddContext(loaderErr, "load config")

	var le *srcloc.Error
	if !errors.As(withCLI, &le) {
		t.Fatalf("expected *srcloc.Error, got %T: %v", withCLI, withCLI)
	}
	if le.Loc == nil {
		t.Fatal("expected non-nil Loc")
	}
	if le.Loc.File != absPath {
		t.Errorf("Loc.File = %q, want %q", le.Loc.File, absPath)
	}

	msg := withCLI.Error()

	// Location appears exactly once.
	if strings.Count(msg, absPath) != 1 {
		t.Errorf("expected file path exactly once in %q", msg)
	}

	// Visible prefix preserves wrapper context in outermost-first order.
	expectedSubstrings := []string{
		"load config",
		"convert " + absPath,
		"parameter \"bignum\"",
	}
	for _, want := range expectedSubstrings {
		if !strings.Contains(msg, want) {
			t.Errorf("missing %q in %q", want, msg)
		}
	}

	// Outermost prefix comes first after the location prefix.
	loadIdx := strings.Index(msg, "load config")
	convertIdx := strings.Index(msg, "convert ")
	paramIdx := strings.Index(msg, "parameter ")
	if !(loadIdx < convertIdx && convertIdx < paramIdx) {
		t.Errorf("prefix ordering wrong in %q (load=%d convert=%d param=%d)",
			msg, loadIdx, convertIdx, paramIdx)
	}

	// Renderer produces snippet with caret on the bignum line.
	rendered := srcloc.NewRenderer().RenderError(withCLI, 2)
	if !strings.Contains(rendered, "^") {
		t.Errorf("expected caret in:\n%s", rendered)
	}
	if !strings.Contains(rendered, "99999999999999999999") {
		t.Errorf("expected source line in:\n%s", rendered)
	}
}
```

- [ ] **Step 9.2: Run test**

Run: `go test ./yaml/ -run TestE2E_LoadConfigPath -v`

Expected: PASS. If prefix ordering fails, double-check that all three `AddContext` calls are using the helper (Task 5.4 step d, Task 5.5 step d, Task 8.1).

- [ ] **Step 9.3: Commit**

```bash
git add yaml/syntax_error_test.go
git commit -m "Test end-to-end load-config rendering preserves prefix order and snippet"
```

---

## Task 10: Cleanup — remove `gopkg.in/yaml.v3`

**Files:**
- Modify: `go.mod`, `go.sum`

- [ ] **Step 10.1: Verify nothing in the repo still imports `gopkg.in/yaml.v3`**

Run: `grep -rn 'gopkg.in/yaml' --include='*.go' .`

Expected: zero hits. If any remain, fix them (likely a missed test file).

- [ ] **Step 10.2: Tidy modules**

Run: `go mod tidy`

Expected: `gopkg.in/yaml.v3` line disappears from `go.mod` and `go.sum`.

- [ ] **Step 10.3: Build and test**

Run: `go build ./...` and `go test ./...`

Expected: PASS.

- [ ] **Step 10.4: Regenerate examples and verify zero diff**

Run: `just gen-examples` (or `go generate ./...` if `just` is unavailable).

Then: `git status` and `git diff --stat`.

Expected: zero changes in `examples/` and `integration/testdata/`. If non-empty diff appears, this is a behavioral divergence from yaml.v3 — read the diff carefully and decide per case (accept goccy behavior in goldens, or add a workaround in DTO).

- [ ] **Step 10.5: Smoke-test the CLI on a syntactically-broken YAML**

```bash
cat > /tmp/badgendi.yaml <<'EOF'
services:
 foo: bar
  baz: qux
EOF
go run ./cmd/gendi --config=/tmp/badgendi.yaml --out=/tmp/out --pkg=di || true
```

Expected stderr contains `/tmp/badgendi.yaml:N:M:` followed by a snippet with `^`. Save the actual output to paste into the PR description.

- [ ] **Step 10.6: Final commit**

```bash
git add go.mod go.sum
git commit -m "Remove gopkg.in/yaml.v3 from dependencies"
```

---

## Acceptance verification (per spec)

After all tasks:

- [ ] `go test ./...` passes.
- [ ] `git diff` after `just gen-examples` is empty.
- [ ] CLI on broken YAML prints `file:line:col: <message>` + snippet with caret (Task 10.5 smoke test).
- [ ] `grep -n 'gopkg.in/yaml' go.mod` returns nothing.
- [ ] `grep -rn 'gopkg.in/yaml' --include='*.go' srcloc/` returns nothing.
- [ ] An IR validation error (e.g. constructor type mismatch) still renders unchanged — pick one IR test that uses snippet rendering and verify by eye that its golden output matches.

---

## Out of scope (per spec — do NOT do these)

- Strict mode / unknown-field rejection.
- Import-chain rendering.
- Color output / `yaml.FormatError`.
- Multi-line caret ranges from goccy `Token.End`.
- `cmd/cli.go:39` (`apply passes`) and `cmd/cli.go:44` (`generate`) wrappers.
- `ir/builder.go:66` and `config.go:23` wrappers.
- `validateDefaults` message localization (lines 283-298 in current parser.go).
