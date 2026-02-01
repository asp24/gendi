# Design Document: Spread Operator for Variadic Arguments

**Status:** Draft
**Date:** 2026-01-11
**Authors:** @anahan

---

## 1. Overview

This document describes the design of a spread operator (`!spread:`) that allows explicit unpacking of slice-typed arguments into variadic parameters in constructor calls.

### 1.1 Motivation

Currently, gendi requires all variadic arguments to be listed individually in YAML configuration:

```go
func NewPipeline(stages ...Stage) *Pipeline
```

```yaml
services:
  pipeline:
    constructor:
      func: NewPipeline
      args:
        - "@stage1"
        - "@stage2"
        - "@stage3"
```

This approach has limitations:
1. **Cannot reuse services returning slices** - If a service returns `[]Stage`, there's no way to pass it to a variadic `...Stage` parameter
2. **Cannot compose with tagged injection** - If a constructor uses variadic parameters, `!tagged:` cannot be used directly
3. **Repetition** - When a collection is already computed elsewhere, it must be manually expanded

### 1.2 Goals

- Provide explicit syntax for unpacking `[]T` into `...T`
- Maintain full explicitness (no implicit magic)
- Support composition with `!tagged:` and service references
- Enable type-safe validation at generation time
- Generate efficient code without runtime overhead

### 1.3 Non-Goals

- Implicit spread behavior (all spreading must be explicit)
- Runtime spread operations
- Spread for non-variadic parameters
- Partial spread (e.g., spreading only some elements)

---

## 2. Design

### 2.1 Syntax

The spread operator uses the prefix `!spread:` followed by any expression that evaluates to a slice type:

```yaml
# Spread a service reference
- "!spread:@stages_collection"

# Spread tagged injection
- "!spread:!tagged:handler"
```

**Note:** Parameters (`%param%`) currently only support primitive types (string, int, float, bool, null), so spread cannot be used with parameters. If slice-typed parameters are added in the future, spread will work automatically.

### 2.2 Semantics

#### 2.2.1 Type Requirements

For `!spread:EXPR` to be valid:
1. `EXPR` must evaluate to a type `[]T` (slice of T)
2. The target constructor parameter must be variadic `...T`
3. The element type `T` must be assignable to the variadic parameter's element type

#### 2.2.2 Evaluation

The spread operator is evaluated in two phases:

**Phase 1: IR Resolution**
- The inner expression is resolved first (e.g., `!tagged:foo` → synthetic service returning `[]Handler`)
- The result type is validated to be a slice
- The target parameter is validated to be variadic
- Element type compatibility is checked

**Phase 2: Code Generation**
- The spread expression is rendered as: `innerExpr...`
- Example: `!spread:@handlers` → `handlers...` in generated code

### 2.3 Examples

#### Example 1: Spread Tagged Services

```go
// Constructor
func NewServer(logger *Logger, handlers ...Handler) *Server
```

```yaml
services:
  server:
    constructor:
      func: NewServer
      args:
        - "@logger"
        - "!spread:!tagged:handler"
```

**Generated code:**
```go
func (c *Container) buildServer() (*Server, error) {
    logger, err := c.getLogger()
    // ...
    handlers, err := c.getTaggedWithHandler()
    // handlers is []Handler
    return NewServer(logger, handlers...), nil
}
```

#### Example 2: Spread Service Reference

```go
// Service returning a slice
func GetAllPlugins() []Plugin

// Constructor accepting variadic
func NewApp(plugins ...Plugin) *App
```

```yaml
services:
  all_plugins:
    constructor:
      func: GetAllPlugins
    shared: true

  app:
    constructor:
      func: NewApp
      args:
        - "!spread:@all_plugins"
```

#### Example 3: Mixed Arguments

```go
func NewPipeline(name string, stages ...Stage) *Pipeline
```

```yaml
services:
  pipeline:
    constructor:
      func: NewPipeline
      args:
        - "my-pipeline"
        - "!spread:!tagged:stage"
```

#### Example 4: Combining Multiple Slices

To combine multiple slices, create a dedicated service:

```go
func NewComplex(handlers ...Handler) *Complex
```

```yaml
services:
  # Combining service
  all_handlers:
    constructor:
      func: CombineHandlers  # func(...Handler) []Handler
      args:
        - "!spread:@primary_handlers"
        - "@fallback_handler"
        - "!spread:@secondary_handlers"
    shared: true

  complex:
    constructor:
      func: NewComplex
      args:
        - "!spread:@all_handlers"  # Single spread, last argument
```

**Note:** Only one `!spread` is allowed per constructor call, and it must be the last argument.

#### Example 5: Without Spread (Regular Slice Parameter)

When constructor accepts a slice (not variadic), spread is not needed:

```go
func NewAggregator(handlers []Handler) *Aggregator
```

```yaml
services:
  aggregator:
    constructor:
      func: NewAggregator
      args:
        - "!tagged:handler"  # No spread needed, returns []Handler
```

---

## 3. Validation Rules

### 3.1 Compile-Time Checks

The following errors must be detected during code generation:

#### 3.1.1 Non-Slice Inner Expression
```yaml
args:
  - "!spread:@logger"  # ERROR: logger is *Logger, not []T
```
**Error:** `!spread: requires slice type, got *Logger`

#### 3.1.2 Non-Variadic Parameter
```yaml
services:
  service:
    constructor:
      func: NewService  # func(handlers []Handler)
      args:
        - "!spread:@handlers"  # ERROR: parameter is []Handler, not ...Handler
```
**Error:** `!spread: can only be used with variadic parameters, got []Handler`

#### 3.1.3 Type Mismatch
```yaml
services:
  service:
    constructor:
      func: NewService  # func(handlers ...Handler)
      args:
        - "!spread:!tagged:middleware"  # []Middleware
```
**Error:** `!spread: element type mismatch: Middleware is not assignable to Handler`

#### 3.1.4 Spread Position

**Rule:** Only one `!spread` is allowed, and it must be the last argument.

This matches Go's variadic semantics where `...` can only be used on the last argument.

```yaml
# Valid: spread is last
args:
  - "@handler1"
  - "@handler2"
  - "!spread:@more_handlers"

# Invalid: multiple spreads
args:
  - "!spread:@first"
  - "!spread:@second"  # ERROR
```
**Error:** `!spread: only one spread allowed per constructor call`

```yaml
# Invalid: spread is not last
args:
  - "!spread:@handlers"
  - "@extra"  # ERROR: arguments after spread
```
**Error:** `!spread: must be the last argument`

### 3.2 Parameter Index Mapping

With spread, parameter index mapping changes:

```go
func NewService(name string, count int, deps ...Dep)
```

```yaml
args:
  - "my-service"                 # arg[0] → param[0] (name)
  - "42"                         # arg[1] → param[1] (count)
  - "!spread:!tagged:dep"        # arg[2] → param[2] (deps), spread
```

The spread argument still maps to parameter index 2, but generates `...` in the call.

---

## 4. Implementation Plan

### 4.1 Phase 1: IR Changes

**File:** `config.go`

Add new argument kind:
```go
type ArgumentKind int

const (
    ArgumentServiceRef ArgumentKind = iota
    ArgumentParameter
    ArgumentTaggedInjection
    ArgumentLiteral
    ArgumentSpread  // NEW
)
```

Add spread field to `Argument`:
```go
type Argument struct {
    Kind  ArgumentKind
    Value string
    Inner *Argument  // For spread, this is the wrapped argument
    // ... existing fields
}
```

**File:** `ir/resolver_argument.go`

Add spread resolution logic:
```go
func (r *argumentResolver) resolve(container *Container, serviceID string, argIndex int, arg string, paramType types.Type) (*Argument, error) {
    // Check for !spread: prefix
    if strings.HasPrefix(arg, "!spread:") {
        return r.resolveSpread(container, serviceID, argIndex, arg, paramType)
    }

    // Existing logic for !tagged:, @service, %param%, etc.
    // ...
}

func (r *argumentResolver) resolveSpread(container *Container, serviceID string, argIndex int, arg string, paramType types.Type) (*Argument, error) {
    // 1. Extract inner expression
    innerExpr := strings.TrimPrefix(arg, "!spread:")

    // 2. Determine expected slice type from variadic param
    //    paramType should be []T (variadic params are represented as slices in go/types)
    sliceType, ok := paramType.(*types.Slice)
    if !ok {
        return nil, fmt.Errorf("service %q arg[%d]: !spread: requires variadic parameter, got %s",
            serviceID, argIndex, paramType)
    }

    // 3. Resolve inner expression with slice type
    innerArg, err := r.resolve(container, serviceID, argIndex, innerExpr, paramType)
    if err != nil {
        return nil, fmt.Errorf("service %q arg[%d]: !spread: %w", serviceID, argIndex, err)
    }

    // 4. Validate that inner resolved to a slice type
    if _, ok := innerArg.Type.(*types.Slice); !ok {
        return nil, fmt.Errorf("service %q arg[%d]: !spread: requires slice type, got %s",
            serviceID, argIndex, innerArg.Type)
    }

    // 5. Return spread argument
    return &Argument{
        Kind:  ArgumentSpread,
        Inner: innerArg,
        Type:  innerArg.Type,  // Type is []T
    }, nil
}
```

**File:** `ir/resolver_constructor.go`

Update validation to check spread position:
```go
func (r *constructorResolver) validateSpreadPosition(cons *Constructor, irCons *IRConstructor) error {
    if !irCons.Variadic {
        return nil  // No variadic, no spread allowed (already checked in resolve)
    }

    // Find all spread arguments
    spreadCount := 0
    lastSpreadIdx := -1
    for i, arg := range irCons.Args {
        if arg.Kind == ArgumentSpread {
            spreadCount++
            lastSpreadIdx = i
        }
    }

    if spreadCount == 0 {
        return nil  // No spread, nothing to check
    }

    // Check that only one spread is present
    if spreadCount > 1 {
        return fmt.Errorf("!spread: only one spread allowed per constructor call")
    }

    // Check that spread is the last argument
    if lastSpreadIdx != len(irCons.Args)-1 {
        return fmt.Errorf("!spread: must be the last argument")
    }

    return nil
}
```

### 4.2 Phase 2: Code Generation

**File:** `generator/arg_builder.go`

Update argument building to handle spread:

```go
func (ab *argBuilder) buildArgument(arg *ir.Argument, paramType types.Type) (stmts []string, expr string, err error) {
    switch arg.Kind {
    case ir.ArgumentSpread:
        // Build the inner expression
        stmts, innerExpr, err := ab.buildArgument(arg.Inner, paramType)
        if err != nil {
            return nil, "", err
        }
        // Add ... to spread it
        return stmts, innerExpr + "...", nil

    // ... existing cases
    }
}
```

**Note:** The `...` suffix is added during code generation, not stored in IR.

### 4.3 Phase 3: Testing

**File:** `generator/generator_test.go`

Add test cases:
1. Spread with tagged injection
2. Spread with service reference
3. Mixed spread and non-spread arguments
4. Error cases: non-slice, non-variadic, multiple spreads, spread not last

**File:** `ir/resolver_test.go`

Add validation tests:
1. Spread position validation (must be last)
2. Spread count validation (only one allowed)
3. Type compatibility checks

### 4.4 Phase 4: Documentation

Update files:
- `doc.md` - Add spread operator to specification
- `AGENTS.md` - Add spread to syntax reference
- `README.md` - Add spread examples
- `examples/` - Add example using spread

---

## 5. Alternative Approaches Considered

### 5.1 Implicit Spread for `!tagged:`

**Idea:** Automatically spread `!tagged:` when used with variadic parameters.

```yaml
args:
  - "!tagged:handler"  # Automatically spreads if parameter is variadic
```

**Rejected because:**
- Violates "explicit over implicit" principle
- Creates ambiguity: is `!tagged:` returning `[]T` or spreading to `...T`?
- Less predictable behavior
- Harder to understand generated code

### 5.2 Spread at Service Level

**Idea:** Mark service definition as "spreadable":

```yaml
services:
  handlers:
    constructor:
      func: GetHandlers
    spread_as_variadic: true

  server:
    constructor:
      func: NewServer
      args:
        - "@handlers"  # Automatically spreads
```

**Rejected because:**
- Mixes service definition with usage context
- Service should be usable as `[]T` in some contexts, as `...T` in others
- Not composable (what about `!tagged:`?)

### 5.3 Special Operator `!tagged-spread:`

**Idea:** Dedicated operator for the common pattern:

```yaml
args:
  - "!tagged-spread:handler"  # Sugar for !spread:!tagged:handler
```

**Deferred because:**
- Adds syntax for a single use case
- Can be added later if `!spread:!tagged:` proves too verbose
- Prefer to start with the minimal, composable primitive

### 5.4 Positional Syntax (Go-like)

**Idea:** Use `...` suffix like Go:

```yaml
args:
  - "@handlers..."
  - "!tagged:middleware..."
```

**Rejected because:**
- Harder to parse (no clear prefix)
- Doesn't follow established `!operator:` pattern
- Less clear for complex expressions

---

## 6. Design Rationale

### 6.1 Why Explicit `!spread:`?

**Decision:** Require explicit `!spread:` operator, even for common `!tagged:` case.

**Rationale:**
1. **Consistency with project philosophy:** AGENTS.md states "Explicit over implicit"
2. **Predictability:** Always clear what type is being passed
3. **No special cases:** Same rule applies to `!tagged:`, `@service`, `%param%`
4. **Better error messages:** Can distinguish between "wrong type" vs "forgot to spread"
5. **Future-proof:** If we ever support slice-of-slice types, no ambiguity

### 6.2 Why Prefix Syntax?

**Decision:** Use `!spread:EXPR` instead of `EXPR...` or postfix syntax.

**Rationale:**
1. **Consistency:** Matches existing `!tagged:` operator style
2. **Composability:** Easy to nest: `!spread:!tagged:foo`
3. **Parsing:** Clear prefix makes YAML parsing straightforward
4. **Familiarity:** Users already understand `!operator:` pattern

### 6.3 Why Restrict to Last Argument?

**Decision:** Enforce that spread must be the last argument in variadic position.

**Rationale:**
1. **Go semantics:** Go only allows `...` on the last argument
2. **Generated code validity:** Ensures we generate syntactically valid Go
3. **Clear error message:** Better to reject at generation time than produce invalid code

---

## 7. Open Questions

### 7.1 Should we validate at IR phase or generation phase?

**Question:** When should spread position validation occur?

**Decision:** At IR resolution phase (Phase 2 of IR building), along with other argument validation. This ensures errors are caught early and consistently.

### 7.2 Should we inline spread arguments?

**Question:** Should we inline spread expressions or always use intermediate variables?

```go
// Option A: Inline
return NewService(c.getHandlers()...)

// Option B: Intermediate variable
handlers, err := c.getHandlers()
if err != nil { return nil, err }
return NewService(handlers...)
```

**Decision:** Follow existing pattern - use intermediate variables for clarity and error handling. Spread doesn't change this.

---

## 8. Migration and Compatibility

### 8.1 Backward Compatibility

This feature is **100% backward compatible**:
- No existing syntax is changed
- No existing behavior is modified
- Purely additive feature

Existing configurations continue to work without any changes.

### 8.2 Future Extensions

This design allows for future extensions:

1. **Spread sugar operator** `!tagged-spread:` could be added as alias
2. **Spread for slices of slices** if needed
3. **Spread from parameters** when/if parameter slices are supported in the future

---

## 9. Success Criteria

This feature is considered successful if:

1. ✅ Users can spread `!tagged:` into variadic parameters
2. ✅ Users can spread service references returning slices
3. ✅ All type mismatches are caught at generation time
4. ✅ Generated code is clean and efficient (no runtime overhead)
5. ✅ Error messages clearly explain validation failures
6. ✅ Documentation and examples cover common use cases
7. ✅ No regression in existing functionality

---

## 10. Timeline

**Phase 1 (IR Changes):** ~2-3 hours
- Add `ArgumentSpread` kind
- Implement spread resolution
- Add position validation

**Phase 2 (Code Generation):** ~1-2 hours
- Update arg_builder to emit `...`
- Test generated code

**Phase 3 (Testing):** ~2-3 hours
- Write unit tests
- Write integration tests
- Create examples

**Phase 4 (Documentation):** ~1-2 hours
- Update specification
- Update README
- Add examples

**Total estimated effort:** ~8-10 hours

---

## 11. References

- **Go Language Spec - Variadic Functions:** https://go.dev/ref/spec#Passing_arguments_to_..._parameters
- **Project Philosophy:** See `AGENTS.md`, section "Key Design Principles"
- **Tagged Injection Implementation:** See `ir/phase_tags_desugar.go`
- **Existing Operator Syntax:** See `!tagged:` in `ir/resolver_argument.go`
