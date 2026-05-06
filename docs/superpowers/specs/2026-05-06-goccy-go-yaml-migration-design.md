# Migration to `github.com/goccy/go-yaml`

**Date:** 2026-05-06
**Status:** Design approved, awaiting implementation plan
**Scope:** YAML parsing layer only

## Problem

When the YAML config file has a syntactic error (bad indentation, unclosed
quote, mapping where a scalar is expected, etc.), the current pipeline returns
a bare error from `gopkg.in/yaml.v3` such as:

```
parse /abs/path/gendi.yaml: yaml: line 5: mapping values are not allowed in this context
```

The error string contains a line number embedded in free text, but the
pipeline does not turn it into a structured location. As a result:

- The CLI prints the message verbatim with no source snippet, no caret, no
  column information.
- The error is not a `*srcloc.Error`, so `srcloc.Renderer.RenderError` falls
  through and prints the raw string.

Validation errors raised later by the IR phases already flow through
`srcloc.Errorf` / `srcloc.WrapError` and render with a snippet plus caret.
Parse-time errors are the outlier.

## Goal

Make YAML parse errors render with the same `file:line:col` + source snippet
+ caret as IR validation errors. No new error styles, no new tooling — the
existing `srcloc.Renderer` already does the rendering; the parser just needs
to produce `*srcloc.Error` values.

## Out of scope

- **Strict mode** (rejecting unknown keys / catching `descorates` typos).
  Tracked separately.
- **Import chain in error output** ("a.yaml imports b.yaml where the error
  is"). Tracked separately.
- **Improving IR validation messages.** They already work; do not touch.
- **Performance.** goccy is comparable; we do not benchmark.

## Approach

Replace `gopkg.in/yaml.v3` with `github.com/goccy/go-yaml` across the
`yaml/` package. Convert anything implementing the `yaml.Error` interface
(syntax errors, type-mismatch errors, unknown-field errors, etc.) to
`*srcloc.Error` at the loader boundary, so the existing `srcloc.Renderer`
picks it up automatically.

We **do not** use goccy's built-in `yaml.FormatError` — its output style
differs from our `srcloc.Renderer`, and we want one rendering style for all
error sources (parser + IR validations).

## Affected packages

| Package | Change |
|---|---|
| `yaml/` | Full migration: DTO `UnmarshalYAML` rewritten on goccy AST API; literal converter rewritten; loader converts goccy errors to `*srcloc.Error`; **all** parser-level and loader-level `fmt.Errorf("...: %w", err)` wrappers that may carry a `*srcloc.Error` (parser.go:45, 82, 93, 193 and config_loader_yaml.go:72, 100) replaced with `srcloc.AddContext` to avoid double-locating. See §3.2 for the full list. |
| `srcloc/` | `NewLocation` loses YAML dependency; signature becomes `(file string, line, col int)`. New helper `srcloc.AddContext(err, format, args...) error` added — see §3. |
| `go.mod` | `gopkg.in/yaml.v3` removed; `github.com/goccy/go-yaml` pinned to **exactly v1.19.2** (no `^` / "or newer"). Reason: this design relies on the `NodeUnmarshaler { UnmarshalYAML(ast.Node) error }` interface and the `yaml.Error` interface (`GetToken`, `GetMessage`), and the new tests assert exact line/column values. Newer goccy versions may shift parser locations, error messages, or AST node types without an intentional migration on our side. Version bumps are a deliberate follow-up, not implicit. |
| `config.go`, `ir/*`, `generator/*` | Untouched. Public types `Service.SourceLoc`, `Argument.SourceLoc`, etc. keep the same `*srcloc.Location` shape. Custom passes do not break. Intermediate `fmt.Errorf("phase %T apply: %w", ...)` (`ir/builder.go:66`) and `fmt.Errorf("pass %q failed: %w", ...)` (`config.go:23`) wrappers stay as is — see §3.2 and Non-goals for the rationale. |
| `cmd/cli.go` | One wrapper (line 34, `load config`) replaced with `srcloc.AddContext`. Lines 39 and 44 (`apply passes`, `generate`) stay as `fmt.Errorf` — they never receive a direct `*srcloc.Error`, only one already wrapped by an intermediate `fmt.Errorf` upstream, so `AddContext` would behave identically to `fmt.Errorf` there. See §3.2. |

## Design

### 1. `srcloc.Location` becomes YAML-library-neutral

**Before:**

```go
// srcloc/location.go
import "gopkg.in/yaml.v3"
func NewLocation(filePath string, node *yaml.Node) *Location { ... }
```

**After:**

```go
// srcloc/location.go — no yaml import
func NewLocation(filePath string, line, column int) *Location {
    if filePath == "" || line < 1 {
        return nil
    }
    return &Location{File: filePath, Line: line, Column: column}
}
```

The `ast.Node` → `(line, col int)` conversion lives in a small unexported
helper in `yaml/`:

```go
// yaml/<helpers>.go
func nodeLoc(n ast.Node) (line, col int) {
    if n == nil {
        return 0, 0
    }
    tok := n.GetToken()
    if tok == nil || tok.Position == nil {
        return 0, 0
    }
    return tok.Position.Line, tok.Position.Column
}
```

**Rationale:** `srcloc` becomes reusable for any future location source
(Go AST, JSON, etc.) and stops leaking a transitive YAML-library dependency
into every package that touches `*srcloc.Location` (which is roughly the
whole project via `config.go`).

**Breaking change:** `srcloc.NewLocation` signature changes. The only
in-repo callers are `yaml/parser.go` and `yaml/config_loader_yaml.go` (both
get migrated). External custom passes are not realistic callers — they
consume `Service.SourceLoc` etc., they do not construct `*srcloc.Location`.

### 2. DTO migration to goccy AST

The current style in `yaml/dto.go` already does manual node walking on top
of `node.Decode(&alias)`. Migration is mostly a 1:1 type swap.

**API mapping:**

| `gopkg.in/yaml.v3` | `github.com/goccy/go-yaml` |
|---|---|
| `*yaml.Node` | `ast.Node` (interface) |
| `node.Decode(&v)` | `yaml.NodeToValue(node, &v)` |
| `node.Kind == yaml.MappingNode` | type assert to `*ast.MappingNode` |
| `node.Content[i], node.Content[i+1]` | `mapping.Values[i].Key`, `.Value` |
| `node.Tag == "!!str"` | type assert to `*ast.StringNode` **or** `*ast.LiteralNode` (no tag) — see "Scalar string compatibility" below |
| `UnmarshalYAML(node *yaml.Node)` | `UnmarshalYAML(node ast.Node)` (`yaml.NodeUnmarshaler`) |
| `node.Line, node.Column` | `node.GetToken().Position.{Line,Column}` |

**`Raw*.Node` field:** type `*yaml.Node` → `ast.Node`. Still tagged
`yaml:"-"`, still purely internal for location tracking.

**`convertLiteral` (`yaml/parser.go:251`):** the tag-based switch
(`!!str/!!int/!!float/!!bool/!!null`) is replaced with a Go type switch
on AST node types.

**Signature change (mandatory).** Today
`convertLiteral(node *yaml.Node) (di.Literal, error)` returns plain
errors and callers wrap them with `fmt.Errorf("parameter %q: %w", ...)`.
After migration, the literal converter must produce node-located errors
itself (overflow, `.inf`/`.nan`, unsupported AST type — see below).
Widen the signature to
`convertLiteral(node ast.Node, filePath string) (di.Literal, error)`
and return `srcloc.Errorf(srcloc.NewLocation(filePath, nodeLoc(node)), ...)`
directly. Callers (`ConvertConfigWithDirAndFile`, `convertArgumentWithFile`)
already have `filePath` in scope.

A `*NodeError` return is **not** an option here: `*NodeError` only
becomes a `*srcloc.Error` inside `toSrclocError`, which today is wired
only to `parseRaw` failures at the loader boundary. `convertLiteral`
runs later, inside `ConvertConfigWithDirAndFile`, whose errors are
wrapped with `fmt.Errorf("convert %s: %w", ...)` in the loader and
never pass through `toSrclocError`. So a `*NodeError` would be flattened
to a plain string and the snippet would be lost.

Callers must also **not** wrap the literal-converter result in a way
that double-locates the error:

- ❌ `fmt.Errorf("parameter %q: %w", name, err)` — `Error()` becomes
  `parameter "foo": file:line:col: msg` and the renderer's prefix logic
  becomes ugly.
- ❌ `srcloc.WrapError(loc, "parameter \"foo\"", err)` where `err` is
  *already* a `*srcloc.Error` with the same `Loc` — current
  `srcloc.Error.Error()` produces
  `file:line:col: parameter "foo": file:line:col: msg`
  (the location is duplicated because nested `%v` triggers the inner
  `Error()`'s own `Loc` formatting).
- ✅ Use the new helper `srcloc.AddContext(err, "parameter %q", name)`
  (see §3) which prepends a contextual prefix into the existing
  `*srcloc.Error.Message` field and **never adds a second location**.

If no extra context is required, return the literal-converter's error
unwrapped.

Per-branch behavior:

- **String:** `*ast.StringNode` *and* `*ast.LiteralNode` (block/folded
  scalars `|` and `>`) both produce `di.NewStringLiteral`. Read
  `.Value` for `StringNode`; for `LiteralNode` read `.Value.Value` (the
  inner `*ast.StringNode`). See "Scalar string compatibility" below.
- **Integer:** `*ast.IntegerNode`. `Value` is `interface{}`; in goccy
  v1.19.2 it is `int64` for in-range positive/negative ints and `uint64`
  for values exceeding `int64` range. The implementation must:
  - accept `int64` directly into `di.NewIntLiteral(int64)`;
  - accept `uint64` only when it fits in `int64` (`v <= math.MaxInt64`);
  - return a structured error otherwise (`"integer value %d does not fit in int64"`),
    located at the node.
  This preserves current behavior, which only ever supported `int64`.
- **Float:** `*ast.FloatNode` only. `*ast.InfinityNode` and `*ast.NanNode`
  are explicitly **rejected** with a node-located error. Rationale:
  yaml.v3's `node.Decode(&v float64)` accepted `.inf`/`.nan` silently;
  no current configs rely on it; rejecting is safer than silently passing
  through values that downstream Go code probably does not expect.
  (If a real config breaks, relax later.)
- **Bool:** `*ast.BoolNode`, read `.Value`.
- **Null:** `*ast.NullNode`, produces `di.NewNullLiteral()`.
- **Default:** node-located error with the AST type name in the message.

No intermediate `node.Decode` calls in any branch.

**`RawArgument` scalar branch (`yaml/dto.go:269`):** the current check
`node.Kind == yaml.ScalarNode && node.Tag == "!!str"` becomes
"`*ast.StringNode` or `*ast.LiteralNode`" (same compatibility note as
above). Semantics preserved: only explicit strings flow into the
prefix-parser path; numbers/bools/nulls flow into `convertLiteral` via
`convertArgumentWithFile`.

**Scalar string compatibility (LiteralNode):** yaml.v3 represents block
(`|`) and folded (`>`) scalars as plain string scalars with `Tag == "!!str"`.
goccy represents them as `*ast.LiteralNode`, distinct from `*ast.StringNode`.
A naïve `node.(*ast.StringNode)` would silently miss them and cause
behavioral drift (e.g., a parameter declared with `value: |\n  hello`
would be rejected). Both `RawArgument.UnmarshalYAML` and
`convertLiteral`'s string branch must accept both node types. Tests
must cover `|` and `>` for parameter values and constructor args.

**`NodeUnmarshaler` is available in the pinned version.** goccy v1.19.2
exposes `UnmarshalYAML(ast.Node) error`; no fallback path is needed.

### 3. Error conversion and prefix-wrapping

This section covers three things, in order: (a) a new helper that lets
us prepend message context without double-locating; (b) the loader's
existing prefix wrappers that must use it; (c) the new `toSrclocError`
that turns goccy errors into `*srcloc.Error`.

#### 3.1 New helper in `srcloc/`

```go
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
// drop the "outer" wrapper. Callers in this codebase must hand
// AddContext the located error directly, which they always do.
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

This is the single approved tool for adding context to a possibly-located
error in this codebase.

**Guarantee (precise):** the helper does not add a *new* `Loc` on top of
an existing `*srcloc.Error`. The contextual prefix is folded into the
existing `Message` field, so `Error()` produces
`file:line:col: <prefix>: <original message>` with the location appearing
exactly once.

**Limitations (intentional, not bugs):**

1. `AddContext` does *not* recurse into `le.Err`. If `le.Err` itself
   wraps another `*srcloc.Error` (a degenerate construction we do not
   produce anywhere today), the chain can still hold multiple locations.
   In current code paths, `le.Err` is always either nil, a plain
   `error`, or a `yaml.Error` already normalized to a plain error inside
   `toSrclocError` (see §3.3).
2. `AddContext` uses a direct `*Error` type assertion, not
   `errors.As`. If a caller passes
   `fmt.Errorf("outer: %w", locErr)`, the `outer` wrapper would be
   silently discarded by an `errors.As`-based implementation. With the
   direct assertion, the wrapped form falls through to the
   `fmt.Errorf("<prefix>: %w", err)` fallback, which preserves both
   `outer` and the inner located-error chain. Callers must therefore
   hand `AddContext` the located error directly (they always do in the
   call sites listed in §3.2).

#### 3.2 Existing `fmt.Errorf("...: %w", err)` wrappers must use it

All existing prefix wrappers that may receive a `*srcloc.Error` from
`convertLiteral`, from a DTO `UnmarshalYAML`, or from `toSrclocError`
must be rewritten as `srcloc.AddContext(...)`. Otherwise the user sees
either ugly double-located prefixes (`convert /abs: /abs:line:col: msg`)
or correctly-rendered snippets but with the original parser context
(`parameter "foo"`, `service "bar"`, `arg[2]`) lost in the formatting.

**Loader (`yaml/config_loader_yaml.go`):**

- line 100: `fmt.Errorf("convert %s: %w", abs, err)` →
  `srcloc.AddContext(err, "convert %s", abs)`. Or, since `abs` is
  already inside the located error's `Loc.File`, the prefix may be
  dropped entirely — pick during implementation by reading actual
  rendered output.
- line 72 (the `parse %s: %w` fallback inside `toSrclocError`, see
  §3.3) →  `srcloc.AddContext(err, "parse %s", file)`.

**Parser (`yaml/parser.go`):**

- line 45: `fmt.Errorf("parameter %q: %w", name, err)` —
  `err` here is the result of `convertLiteral` (parameter value), now a
  `*srcloc.Error`. Replace with
  `srcloc.AddContext(err, "parameter %q", name)`.
- line 82: `fmt.Errorf("_default: %w", err)` — wraps `validateDefaults`,
  which today returns plain errors, but the spec for `validateDefaults`
  itself is out of scope here — leave the wrapper as
  `srcloc.AddContext` for forward compatibility with no behavior change.
- line 93: `fmt.Errorf("service %q: %w", name, err)` — wraps
  `convertServiceWithPackageAndFile`, whose internals now produce
  `*srcloc.Error` via `convertArgumentWithFile` → `convertLiteral`.
  Replace with `srcloc.AddContext(err, "service %q", name)`.
- line 193: `fmt.Errorf("arg[%d]: %w", i, err)` — wraps
  `convertArgumentWithFile`, which calls `convertLiteral`. Replace with
  `srcloc.AddContext(err, "arg[%d]", i)`.

**Terminal `fmt.Errorf` calls in `parser.go` (lines 41, 248, 276,
283–298):** these create *new* errors rather than wrap. They should
become `srcloc.Errorf(srcloc.NewLocation(filePath, nodeLoc(node)), ...)`
when a relevant node is in scope (e.g. line 41 — the parameter mapping
node; line 276 — the literal node; lines 283–298 — the `_default`
service node). This is the "Improving validation messages" follow-up's
territory but trivially in-scope here for any line that already had a
node in lexical scope. Implementer judgment: convert opportunistically;
do not add new node tracking just for this.

**CLI (`cmd/cli.go`):**

- line 34: `fmt.Errorf("load config: %w", err)` — receives a direct
  `*srcloc.Error` from `toSrclocError` (or from the parser-level
  `AddContext` chain above it). Replace with
  `srcloc.AddContext(err, "load config")`.

**CLI wrappers we deliberately leave alone:**

- line 39: `fmt.Errorf("apply passes: %w", err)` — `di.ApplyPasses`
  in `config.go:23` already wraps the underlying error as
  `fmt.Errorf("pass %q failed: %w", name, err)`. Whatever a custom pass
  returned (located or not) is no longer a direct `*srcloc.Error` by
  the time it reaches `cli.go:39`. `AddContext`'s direct type
  assertion would always fall through to its
  `fmt.Errorf("apply passes: %w", err)` fallback — i.e. identical to
  the existing `fmt.Errorf`. No fold benefit, no harm; leaving
  `fmt.Errorf` keeps the diff minimal and the intent honest.
- line 44: `fmt.Errorf("generate: %w", err)` — same situation:
  `ir/builder.go:66` wraps phase errors as
  `fmt.Errorf("phase %T apply: %w", phase, err)`, so by the time the
  located `*srcloc.Error` reaches `cli.go:44` it is no longer direct.
  `AddContext` would also fall through here. Leave as `fmt.Errorf`.

**Consequence and follow-up:** for `generate` and `apply passes`
chains the rendered output still contains the location exactly once
(only one `*srcloc.Error` in the chain) and the snippet is still
rendered (`Renderer` uses `errors.As` to find the inner located
error). The visible prefix is just longer:
`generate: phase X apply: file:line:col: <inner message>`. This is
status-quo behavior — no regression. If a future change wants the
prefix to fold cleanly the way `load config` does, the right fix is
to replace `fmt.Errorf` in **both** `ir/builder.go:66` and
`config.go:23` with `srcloc.AddContext`. That is **out of scope** for
this migration (see Non-goals). Capturing it here so the follow-up
PR has a clear starting point.

**Untouched `fmt.Errorf` wrappers** (no `*srcloc.Error` ever flows
through them today): `config_loader_yaml.go:55, 81, 86, 163`,
`cmd/cli.go:29` (Finalize validates CLI flags), `cmd/cli.go:49`
(WriteTargetFile is plain file I/O). Leave as is.

#### 3.3 `toSrclocError` — converting goccy errors

After the swap, errors out of `parseRaw` fall into three buckets:

1. **`*NodeError`** — our DTO-level validation (`import path is required`,
   `service must be a mapping or alias`, `tag name is required`).
   Pre-existing; only the field type of `NodeError.Node` changes.
2. **Anything implementing `yaml.Error`** — goccy v1.19.2 defines a
   public `yaml.Error` interface with `GetToken() *token.Token` and
   `GetMessage() string`. It is implemented by `*yaml.SyntaxError` *and*
   by separate types for type mismatch, integer overflow, duplicate key,
   unknown field, and unexpected node. Catching only `*yaml.SyntaxError`
   would let several real decode failures fall through to the bare
   `fmt.Errorf` path, losing the snippet — and worse, possibly letting
   goccy's own formatter leak into `Error()` output and break our
   single-style invariant. **Convert via the interface, not the concrete
   type.**
3. **Anything else** — wrapped via `srcloc.AddContext(err, "parse %s", file)`.
   Should be unreachable in practice, but kept as a safe fallback that
   does not double-locate if the residual error happens to be a
   `*srcloc.Error`.

**Conversion happens at the loader boundary**, replacing the current block
in `yaml/config_loader_yaml.go:60-73`:

```go
raw, err := l.parseRaw(data)
if err != nil {
    return nil, l.toSrclocError(abs, err)
}
```

```go
func (l *ConfigLoaderYaml) toSrclocError(file string, err error) error {
    var ne *NodeError
    if errors.As(err, &ne) {
        line, col := nodeLoc(ne.Node)
        loc := srcloc.NewLocation(file, line, col)

        // ne.Err may itself wrap a goccy yaml.Error (e.g. NodeToValue
        // failed inside a custom UnmarshalYAML). srcloc.WrapError would
        // call ne.Err.Error(), which routes through goccy's own
        // formatter and breaks the single-style invariant. Normalize:
        // strip the wrapped yaml.Error down to its plain message.
        wrapped := ne.Err
        var inner yaml.Error
        if errors.As(wrapped, &inner) {
            wrapped = errors.New(inner.GetMessage())
        }
        return srcloc.WrapError(loc, ne.Msg, wrapped)
    }

    var ye yaml.Error
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

Once converted, `cmd/cli.go:77`'s existing
`srcloc.NewRenderer().RenderError(err, 4)` renders snippet + caret with no
further changes.

**`yaml.FormatError` is intentionally not used.** Reason: its output style
differs from `srcloc.Renderer` (different colors, different snippet
shape), and we want a single style across parser and IR errors.

### 4. Defensive guards

- `ye.GetToken() == nil` or `tok.Position == nil` → `Loc` is `nil`;
  Renderer prints text without snippet (existing behavior for locationless
  errors).
- `nodeLoc(nil)` → returns `(0, 0)`; `NewLocation` returns `nil`.
- Empty input file → goccy returns either nil error with empty doc list or
  a `yaml.Error`; both paths are safe.

## Testing

### Mechanical updates to existing tests

- `yaml/parser_test.go`, `yaml/config_loader_yaml_test.go`,
  `yaml/node_error_test.go` — anywhere a `*yaml.Node` is constructed by
  hand, replace with `parser.ParseBytes(...)`-based helpers (introduce
  `mustParseNode(t, "...")` if convenient).
- `srcloc/location_test.go` — `NewLocation` now takes `(file, line, col)`;
  tests get simpler.
- Tests asserting our own validation messages (`import path is required`)
  do not change.
- If any test asserts the literal text of a yaml.v3 syntax error, rewrite
  the assertion to check `errors.As(&srcloc.Error)` + non-empty message
  rather than exact string. (Quick grep suggests there are none today;
  verify during implementation.)

### New tests

In `yaml/` (table-driven, possibly a new `syntax_error_test.go`):

1. **Bad YAML → `*srcloc.Error` with exact location.**
   Inputs: missing colon, bad indent, unclosed quote, mapping where scalar
   expected. For each: assert `errors.As(err, &srcLocErr)`, file path
   matches, and `Loc.Line` / `Loc.Column` equal exact expected values
   (not just non-zero — exact assertions catch off-by-one regressions
   when goccy's reporting changes between versions).
2. **Non-`SyntaxError` `yaml.Error` → `*srcloc.Error`.** Triggered by
   a `yaml.NodeToValue` decode failure on a typed Go field. Good
   candidates:
   - `decoration_priority: "abc"` (string for an `int` field);
   - `exclude: 5` under an import (scalar where a `[]string` is
     expected);
   - `shared: "yes"` (string for `*bool`).
   Note: integer overflow inside a `parameters: { foo: { value: ... } }`
   block does *not* fit here, because parameter values are captured as
   `ast.Node` and only converted later by `convertLiteral` — that path
   is covered separately by the `convertLiteral`-overflow unit test.
   The goal of this test is specifically the `NodeToValue` (Go-field
   decode) error path, to assert that the loader catches `yaml.Error`
   via the interface, not the concrete `*SyntaxError`. Two cases —
   one that triggers an inner-`UnmarshalYAML` `NodeError`-wrapping-
   `yaml.Error` (validating the normalization in `toSrclocError`) and
   one that bubbles `yaml.Error` directly out of the top-level
   `yaml.Unmarshal` call.
3. **Empty file / `yaml.Error` without token** — does not panic; returns
   either nil config or an error with `Loc == nil`.
4. **Block (`|`) and folded (`>`) scalars.** Four cases — the cross
   product of {parameter value, constructor arg} × {`|`, `>`}. Each
   must decode to exactly the string the equivalent quoted scalar would
   produce (block: literal newlines preserved; folded: single newlines
   collapsed to spaces). Guards against `LiteralNode`-vs-`StringNode`
   regressions and against accidentally treating only one of the two
   block styles.
5. **`convertLiteral` error paths.** Each must produce a `*srcloc.Error`
   with the location of the offending node:
   - integer overflow (`value: 99999999999999999999`);
   - `value: .inf`, `value: .nan` (rejected — explicit choice in §2);
   - unsupported AST type (e.g. mapping where a literal is expected).
   These exercise the signature change documented in §2 and ensure the
   wrapping path actually carries a `Loc`.
6. **End-to-end render** — one golden test: take a known-bad YAML, call
   `Renderer.RenderError(err, 2)`, assert the rendered string contains
   the expected line, caret column, and message. This catches the regress
   where the parser sets `Loc` but the Renderer fails to render.

In `srcloc/` (new file `addcontext_test.go`):

7. **`AddContext` unit tests** — table-driven, covering each
   documented branch and limitation:
   - **nil err** → returns nil.
   - **direct `*Error`** → returns a new `*Error` with same `Loc`,
     `Err` preserved, `Message == "<prefix>: <old message>"`. Assert
     `Error()` produces a single `file:line:col:` prefix.
   - **plain error** (`errors.New("boom")`) → fallback path; result
     wraps via `fmt.Errorf`, `errors.Is(result, original)` is true,
     `Error()` is `"<prefix>: boom"` with no location.
   - **wrapped located error** (`fmt.Errorf("outer: %w", locErr)`) →
     fallback path (NOT folded); `Error()` contains both `outer` and
     the inner `file:line:col`; `errors.As(result, &*Error)` still
     finds the inner located error so the renderer still produces a
     snippet. Documents the §3.1 Limitation #2.
   - **prefix with format args** (`AddContext(le, "service %q", "foo")`)
     → message is `service "foo": <old>`; verifies `fmt.Sprintf`
     formatting works as expected.

In `yaml/` or a new `cmd/cli_render_test.go`:

8. **End-to-end load-config path with parser/loader/CLI wrappers in
   the chain** — covers the only chain where prefix folding is
   in scope (see §3.2). Feeds a YAML triggering a `convertLiteral`
   error (e.g. integer overflow inside a parameter value); simulates
   the wrapper stack `convertLiteral → parser:45 AddContext("parameter %q") →
   loader:100 AddContext("convert %s") → cli:34 AddContext("load config")`;
   asserts:
   - the resulting error is itself a `*srcloc.Error` (not just
     reachable via `errors.As`);
   - `Error()` contains the location exactly **once**, at the very
     start (no `/abs/path: /abs/path:` duplication);
   - the visible prefix is `file:line:col: load config: convert /abs/path:
     parameter "foo": <inner message>` — exact ordering from outermost
     `AddContext` inward;
   - `Renderer.RenderError(err, 2)` produces a snippet with the caret
     on the expected column.
   This is the regression test that catches any future addition of a
   `fmt.Errorf("...: %w", err)` wrapper *anywhere on the load-config
   path*. The `generate` and `apply passes` paths are explicit
   non-goals (Non-goals section), so this test does not cover them.

### Acceptance criteria

1. `go test ./...` passes.
2. `git diff` after `just gen-examples` is empty — i.e., goccy parses
   every existing config the same way yaml.v3 did.
3. Feeding a syntactically broken YAML to the CLI prints
   `file:line:col: <message>` plus a snippet with a caret. (Manual smoke
   test; one example documented in the PR.)
4. `go.mod` no longer contains `gopkg.in/yaml.v3`; contains
   `github.com/goccy/go-yaml`.
5. `srcloc/` has no import of any YAML library.
6. Existing IR validation errors render unchanged (regression check —
   pick one IR test that uses snippet rendering).

## Risks

- **Behavioral divergence between yaml.v3 and goccy.** Most likely culprits:
  block/folded scalars (handled explicitly above), `.inf`/`.nan` floats
  (rejected explicitly above), `int64`-vs-`uint64` integer values (handled
  explicitly above), unquoted scalar edge cases, comment handling, or
  duplicate-key behavior. Mitigation: criterion 2 below (empty `git diff`
  after regenerating examples) catches divergence early; if found, decide
  per case — accept goccy behavior or add a workaround in the DTO layer.
- **goccy error messages differ from yaml.v3.** Acceptable — they are
  generally better and we do not assert them exactly.

## Non-goals (explicit)

- Strict mode for unknown keys.
- Import chain in error rendering.
- Color output.
- Multi-line caret ranges (using goccy's Start/End token positions).
- **Prefix folding for the `generate` and `apply passes` paths**
  (i.e. `ir/builder.go:66` and `config.go:23` `fmt.Errorf` wrappers).
  These chains keep the existing `phase X apply: ...` /
  `pass "X" failed: ...` intermediate prefixes. Snippets still render.
  Documented in §3.2 with an exact pointer to the two lines a follow-up
  PR would touch.

All of these are possible follow-ups once the migration lands.
