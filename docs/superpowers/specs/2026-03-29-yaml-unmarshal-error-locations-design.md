# YAML Unmarshal Error Locations

## Problem

Errors during `UnmarshalYAML` in `yaml/dto.go` lack source location information. When a user has a misconfigured YAML file (e.g., missing tag name, wrong node type), the error message doesn't indicate where in the file the problem is. The existing `srcloc` rendering pipeline (file:line:column + code snippet with caret) only works for errors produced after YAML parsing, during config validation in `parser.go`.

## Goal

Wrap errors from `UnmarshalYAML` methods with YAML node location info so they flow through the `srcloc.Renderer.RenderError` pipeline and display file:line:column with a code snippet and caret pointer.

## Design

### `NodeError` type (new file: `yaml/node_error.go`)

A package-private error type that carries a `*yaml.Node` (line/column) without a file path:

```go
type NodeError struct {
    Node *yaml.Node
    Msg  string
    Err  error // optional wrapped error
}

func (e *NodeError) Error() string
func (e *NodeError) Unwrap() error

func nodeErrorf(node *yaml.Node, format string, args ...any) *NodeError
func wrapNodeError(node *yaml.Node, msg string, err error) *NodeError
```

`UnmarshalYAML` methods use `nodeErrorf`/`wrapNodeError` instead of `fmt.Errorf`:

```go
// Before:
return fmt.Errorf("tag name is required")
// After:
return nodeErrorf(node, "tag name is required")
```

### Enrichment in `config_loader_yaml.go`

In `loadRecursive`, after `parseRaw` fails, walk the error chain for a `NodeError` and convert it to a `srcloc.Error` by combining the node's line/column with the file path:

```go
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

This is the single enrichment point where the file path meets node errors.

Non-`NodeError` errors (e.g., yaml.v3 `TypeError`) continue through the existing `fmt.Errorf("parse %s: %w")` path unchanged.

### Error sites converted

All custom validation errors in `dto.go` `UnmarshalYAML` methods that have access to a `*yaml.Node`:

| Method | Error message | Node source |
|--------|--------------|-------------|
| `RawImport.UnmarshalYAML` | "import path is required" | `node` |
| `RawImport.UnmarshalYAML` | "import must be a string or mapping" | `node` |
| `RawServiceTag.UnmarshalYAML` | "tag name is required" | `node` |
| `RawServiceTag.UnmarshalYAML` | "tag must be a string or mapping" | `node` |
| `RawServiceTag.UnmarshalYAML` | "failed to decode tag name" | `node` |
| `RawServiceTag.UnmarshalYAML` | "failed to decode tag attribute" | child `valueNode` |
| `RawService.UnmarshalYAML` | "service must be a mapping or alias" | `node` |

Passthrough errors from `node.Decode(...)` (yaml.v3 internal) are not wrapped in `NodeError`.

### What does NOT change

- `parser.go` validation errors already use `srcloc` via stored `Node` fields -- no changes needed.
- `srcloc` package itself is unchanged.
- Existing error rendering via `Renderer.RenderError` is unchanged -- it already handles `*srcloc.Error`.

## Testing

### Unit tests (`yaml/node_error_test.go`)

- `NodeError.Error()` formatting with and without wrapped error
- `NodeError.Unwrap()` chain works with `errors.As`

### Integration tests (in `config_loader_yaml` test file)

- Feed malformed YAML that triggers each error site
- Assert the returned error is a `*srcloc.Error` with correct `Line`, `Column`, and `File`
- Verify `Renderer.RenderError` produces the expected snippet with caret
