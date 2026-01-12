# Technical Specification  
## Go DI Container Code Generator (Symfony DI–Inspired)

---

## 1. Purpose and Scope

The goal of this project is to develop a Go library and a code generator that produces a **statically typed Dependency Injection (DI) container** from declarative YAML configuration, partially inspired by the Symfony Dependency Injection Container.

Key objectives:

- Generate Go code without runtime reflection.
- Generate a dedicated getter method for each service with a **concrete Go type** in its signature.
- Explicit dependency configuration (no autowiring).
- Support:
  - service decorators,
  - tagged services,
  - configuration modification during code generation,
  - configuration composition via imports.
- Detect and report configuration and type errors **at generation time**, not at runtime.

---

## 2. Terminology

| Term | Definition |
|-----|------------|
| Service ID | String identifier of a service in configuration |
| Constructor | Go function or method that creates a service instance |
| Service Type | Type returned by the service getter |
| Shared service | Singleton service |
| Non-shared service | Factory service |
| Decorator | Service that wraps another service |
| Tag | Label assigned to a service |
| Tagged injection | Injection of a collection of services by tag |
| Compiler pass | Programmatic modification of configuration before generation |

---

## 3. In Scope / Out of Scope

### 3.1 In Scope

- YAML-based service configuration
- Go code generation for DI container
- Strict static type validation
- Decorators and decorator chains
- Tagged services
- Configuration imports
- `go generate` workflows

### 3.2 Out of Scope

- Autowiring
- Runtime DI / service locator
- Reflection-based resolution
- Expression language or env interpolation
- Hot-reload of configuration

---

## 4. Architecture Overview

### 4.1 Components

1. **CLI Generator**
   - Parses YAML
   - Resolves imports
   - Applies compiler passes
   - Analyzes Go code (`go/packages`)
   - Generates container code

2. **Runtime Package (Minimal)**
   - Shared error types
   - Optional helper types

3. **Generated Container**
   - `Container` struct
   - Typed getter methods
   - Lazy initialization for shared services
   - No reflection

---

## 5. CLI Generator

### 5.1 Command

```

di-gen

````

### 5.2 Arguments

| Flag | Description |
|----|------------|
| `--config` | Root YAML configuration file |
| `--out` | Output directory or file |
| `--pkg` | Go package name |
| `--container` | Container struct name |
| `--strict` | Enable strict validation (default: true) |
| `--build-tags` | Go build tags |
| `--verbose` | Verbose logging |

### 5.3 go:generate Support

```go
//go:generate di-gen --config=di.yaml --out=./internal/di --pkg=di
````

---

## 6. YAML Configuration

### 6.1 Root Structure

```yaml
imports: []
parameters: {}
tags: {}
services: {}
```

---

## 6.2 Configuration Imports

### 6.2.1 Purpose

Imports allow:

* configuration decomposition,
* reuse across modules,
* overriding of services and parameters.

### 6.2.2 Format

```yaml
imports:
  - ./services/db.yaml
  - ./services/payments.yaml
  - ./services/*.yaml
  - path: github.com/acme/billing/config/*.yaml
```

### 6.2.3 Rules

* Imports are processed **in declaration order**.
* Relative paths are resolved relative to the importing file.
* Recursive imports are allowed.
* Cyclic imports are forbidden.
* Later definitions override earlier ones.
* Service overriding is allowed.
* Imports can be a string path or a mapping with `path`.
* Module imports resolve to `gendi.yaml`/`gendi.yml` at module root when no file is provided.
* Glob patterns are supported; matches are expanded in lexicographic order.

---

## 6.3 Services

### 6.3.1 General Format

```yaml
services:
  service.id:
    type: "pkg.Type"              # optional
    constructor:
      func: "pkg.NewService"      # or method
      args: []
    shared: true
    public: true
    decorates: other.service
    decoration_priority: 10
    tags:
      - name: tag.name
        priority: 10              # any field except 'name' becomes an attribute
        custom_field: value
```

**Service Tags:**

Tags are specified as a list of mappings. Each tag must have a `name` field. All other fields are automatically treated as tag attributes:

```yaml
tags:
  - name: handler.http
    priority: 10
    path: /api/users

  - name: marker.tag           # Tag with no attributes (only name)
```

This is equivalent to the old verbose syntax:

```yaml
tags:
  - name: handler.http
    attributes:
      priority: 10
      path: /api/users
```

---

### 6.3.2 Service Defaults

The `_default` key allows setting default values for `shared` and `public` fields across all services:

```yaml
services:
  _default:
    shared: true      # All services are shared by default
    public: false     # All services are private by default

  logger:
    type: "*app.Logger"
    constructor:
      func: app.NewLogger
    public: true      # Overrides default: make this one public

  cache:
    type: "*app.Cache"
    constructor:
      func: app.NewCache
    # Inherits: shared=true, public=false
```

Rules:
* Only `shared` and `public` fields are allowed in `_default`
* Explicit values in individual services always override defaults
* Other fields (type, constructor, alias, decorates, tags) are forbidden in `_default`
* If a field is not set in `_default` or the service, the standard default applies (shared defaults to true, public defaults to false)

---

### 6.3.3 `type` Field (Optional)

* If omitted, the service type is **inferred from the constructor return type**.
* If present, it is treated as a **contract**.
* The generator must verify **strict equality** with the inferred constructor type.
* Mismatch results in a generation error.

---

### 6.3.4 Constructor

Supported forms:

```yaml
constructor:
  func: "module/pkg.NewX"
```

or

```yaml
constructor:
  method: "@serviceId.Method"
```

---

### 6.3.4.1 The `$this` Package Token

The special token `$this` can be used in service `type`, constructor `func`, and `method` fields to reference the Go package where the configuration file is located.

**Purpose:**

* Eliminates repetitive package paths in service definitions
* Automatically resolves to the package containing the config file
* Each imported config file has its own `$this` context

**Usage:**

```yaml
services:
  logger:
    type: "*$this.Logger"       # Resolves to current package
    constructor:
      func: "$this.NewLogger"   # Resolves to current package
```

**Resolution:**

* `$this` is resolved by:
  1. Finding the directory containing the config file
  2. Walking up to find `go.mod`
  3. Computing the full Go package path relative to module root
* For a config at `/home/user/myapp/services/config.yaml` in module `github.com/user/myapp`:
  * `$this` resolves to `github.com/user/myapp/services`
  * `$this.NewLogger` becomes `github.com/user/myapp/services.NewLogger`

**Rules:**

* For `type` field: `$this.` can appear anywhere in the type (supports `*$this.T`, `[]$this.T`, `map[K]$this.V`, etc.)
* For `func` and `method` fields: `$this` must appear at the start of the path
* Each imported file has its own `$this` context based on its location
* If package resolution fails, `$this` remains unchanged and will cause a generation error if the symbol is not found

**Example:**

```yaml
# File: /myapp/config.yaml
# Module: github.com/acme/myapp

services:
  logger:
    type: "*$this.Logger"           # → *github.com/acme/myapp.Logger
    constructor:
      func: "$this.NewLogger"       # → github.com/acme/myapp.NewLogger

  cache:
    type: "*$this.Cache"            # → *github.com/acme/myapp.Cache
    constructor:
      func: "$this.NewCache"        # → github.com/acme/myapp.NewCache
```

---

### 6.3.5 Allowed Constructor Signatures

A constructor must return **exactly one** of the following:

1. `T`
2. `(T, error)`
3. `*T`
4. `(*T, error)`

Where `T` is:

* a named concrete type,
* **not** `any`,
* **not** `interface{}`,
* **not** a type parameter.

Violation results in a generation error.

---

### 6.3.6 Constructor Arguments

| Syntax                    | Meaning                                  |
| ------------------------- | ---------------------------------------- |
| `@service.id`             | Reference to another service             |
| `@.inner`                 | Inner service in a decorator             |
| `%param.name%`            | Parameter                                |
| `!tagged:tag.name`        | Tagged injection                         |
| `!spread:@service`        | Spread slice into variadic parameters    |
| `!spread:!tagged:tag`     | Spread tagged collection into variadic   |
| literal                   | YAML scalar literal                      |

Argument count and types are strictly validated.

**Spread Operator:**

The `!spread:` operator unpacks slice values into variadic parameters using Go's `...` syntax:

```yaml
services:
  all_handlers:
    constructor:
      func: "app.GetHandlers"  # Returns []Handler

  server:
    constructor:
      func: "app.NewServer"     # NewServer(handlers ...Handler)
      args:
        - "!spread:@all_handlers"  # Unpacks []Handler into ...Handler
```

Rules:
- Only one spread allowed per constructor
- Spread must be the last argument
- Inner value must be a slice type
- Target parameter must be variadic
- Works with both service references (`!spread:@service`) and tagged injection (`!spread:!tagged:tag`)

---

## 6.4 Parameters

```yaml
parameters:
  dsn:
    type: string
    value: "postgres://..."
```

Rules:

* `type` and `value` are mandatory.
* Type is used for Go code generation.
* Parameters are immutable.
* Parameters are resolved at runtime via a `parameters.Provider` passed to the container.

Generated containers include a constructor that accepts a provider. When no provider is passed,
the container falls back to a map-backed provider built from the YAML values.

Supported parameter types:

* `string`
* `int`
* `bool`
* `float64`
* `time.Duration`

Notes:

* `time.Duration` values are defined as YAML strings (for example, `"1s"`).
* Generated code calls `Provider.GetDuration` for `time.Duration` parameters.

---

## 6.5 Service Aliases

Aliases let you reference another service by ID without defining a constructor.

```yaml
services:
  logger:
    constructor:
      func: "example.com/app.NewLogger"
    public: true
  logger_alias: "@logger"
```

You can also use the expanded form to set flags like `public`:

```yaml
services:
  logger_alias:
    alias: "logger"
    public: true
```

---

## 6.6 Tags

### 6.6.1 Tag Declaration

Tags can be declared explicitly or created implicitly when referenced by services.

**Explicit declaration:**

```yaml
tags:
  payment.provider:
    element_type: "example.com/app/payments.Provider"
    sort_by: priority
    public: true
    auto: false
```

**Implicit creation (no declaration needed):**

```yaml
services:
  provider.stripe:
    constructor:
      func: "app.NewStripeProvider"
    tags:
      - name: payment.provider
        priority: 100
```

Only explicitly declared tags can be public.

The following fields are supported on tag declarations:
* `element_type` (optional, required for public or auto tags)
* `sort_by` (optional)
* `public` (optional)
* `auto` (optional)

### 6.6.2 Element Type Inference

* `element_type` is **optional** in tag declarations.
* When `public: true`, `element_type` is required.
* When omitted, the element type is **inferred from constructor arguments** that use the tag via `!tagged:tag.name`.
* The inferred type is the slice element type of the constructor parameter.
* If multiple constructors use the same tag, all must have compatible element types.

**Example:**

```yaml
# No tags section needed - element_type is inferred from NewService's parameter
services:
  provider.stripe:
    constructor:
      func: "app.NewStripeProvider"
    tags:
      - name: payment.provider

  service:
    constructor:
      func: "app.NewService"
      args:
        - "!tagged:payment.provider"  # Parameter type []Provider infers element_type
```

### 6.6.3 Compatibility Validation

For each service with a tag:

* `ServiceType` must be assignable to `element_type` (whether declared or inferred).
* Otherwise, generation fails.

### 6.6.4 Tagged Injection

* Generates a value of type `[]element_type`.
* Sorting:

  * by `priority` attribute if enabled,
  * descending order (`100` before `10`).

---

### 6.6.5 Public Tag Getters

Declared tags can expose a public getter when `public: true` is set:

```yaml
tags:
  payment.provider:
    element_type: "example.com/app/payments.Provider"
    public: true
```

Generated signature:

```go
func (c *Container) GetTaggedWithPaymentProvider() ([]payments.Provider, error)
```

The getter returns all tagged services in the same order as tagged injection.

---

### 6.6.6 Auto Tagging

Auto tagging can be enabled on a tag by setting `auto: true`. The generator
will add the tag to every service whose **final** type implements the tag's
`element_type`.

Rules:
* `element_type` is required and must be an interface type.
* `auto: true` cannot be combined with `sort_by`.
* Auto tagging runs after decorator expansion; only the outermost decorator
  is considered for tagging.
* Alias and `.inner` services are excluded.

Example:

```yaml
tags:
  handler:
    element_type: "github.com/myapp.Handler"
    auto: true
```

See `doc/auto-tagging.md` for full rules and details.

---

## 7. Decorators (Compile-Time)

### 7.1 Model

```yaml
service.decorator:
  decorates: base.service
  decoration_priority: 20
```

### 7.2 Rules

* The base service getter returns the **outermost decorator**.
* `@.inner` is only available inside decorators.
* Decorators are ordered by `decoration_priority`.

### 7.3 Type Validation

* Decorator type must be compatible with the decorated service type.
* Otherwise, generation fails.

---

## 8. Compiler Passes

### 8.1 Purpose

Compiler passes allow:

* dynamic service registration,
* decorator injection,
* argument modification,
* infrastructure wiring.

### 8.2 API

```go
type Pass interface {
    Name() string
    Process(cfg *Config) (*Config, error)
}

// Apply passes before generation
cfg, err := di.ApplyPasses(cfg, []di.Pass{
    mypass.New(),
})
```

Passes mutate the config and return it for chaining.

### 8.3 Execution Order

1. YAML loading + imports
2. Compiler passes (`di.ApplyPasses`)
3. Generator options finalization
4. Code generation

### 8.4 Usage Example

```go
// Load config
cfg, err := yaml.LoadConfig("gendi.yaml")

// Apply passes (config layer)
cfg, err = di.ApplyPasses(cfg, []di.Pass{
    myCustomPass{},
})

// Configure generator
opts := generator.Options{
    Out:     "./internal/di",
    Package: "di",
}
opts.Finalize()

// Generate
gen := generator.New(cfg, opts)
code, err := gen.Generate()
```

---

## 9. Container Generation

### 9.1 Structure

```go
type Container struct {
    // private fields
}
```

### 9.2 Getter Methods

* `GetX() T`
* `GetX() (T, error)` — if constructor returns error

All getters are strictly typed.

### 9.3 Shared vs Non-shared

* Shared → lazy singleton
* Non-shared → new instance per call

---

## 10. Error Reporting

The generator must emit diagnostic errors containing:

* service ID,
* configuration field,
* expected vs actual type,
* dependency chain if applicable.

Example:

```
service "payments":
  constructor "NewService":
  arg[0]: expected []payments.Provider, got []any
```

---

## 11. Circular Dependency Detection

* Cycles are detected **at generation time**.
* In strict mode, generation fails.
* Error messages must include the dependency trace.

---

## 12. Testing Requirements

### 12.1 Unit Tests

* YAML parsing
* Import resolution
* Constructor validation
* Type inference

### 12.2 Integration Tests

* Golden file comparisons
* Decorators
* Tagged injection
* Imports and overrides

---

## 13. Deliverables

* CLI generator
* Runtime package
* Documentation
* Examples
* Test suite

---

## 14. Fixed Decisions (v1)

1. `services.type` is optional
2. Tagged injection produces `[]T` only
3. `tags.element_type` is optional (required when `public: true` or `auto: true`)
4. Strict typing, no `any`
5. Errors detected at generation time
