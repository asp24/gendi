# Configuration Reference

Complete reference for gendi YAML configuration files.

## Schema Validation

Add this line at the top of your YAML files for editor autocomplete and validation:

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/asp24/gendi/master/gendi.schema.json
```

Supported editors:
- **VS Code**: Install [YAML extension](https://marketplace.visualstudio.com/items?itemName=redhat.vscode-yaml)
- **IntelliJ IDEA**: Built-in YAML support
- **Vim/Neovim**: Use [yaml-language-server](https://github.com/redhat-developer/yaml-language-server)

For local schema validation:
```yaml
# yaml-language-server: $schema=./gendi.schema.json
```

## Table of Contents

- [Parameters](#parameters)
- [Services](#services)
- [Tags](#tags)
- [Imports](#imports)
- [Special Tokens](#special-tokens)
- [Argument Syntax](#argument-syntax)

## Parameters

Parameters are typed configuration values injected using `%name%` syntax.

### Parameter Definition

```yaml
parameters:
  app_name:
    type: string
    value: "MyApp"

  port:
    type: int
    value: 8080

  timeout:
    type: time.Duration
    value: "30s"

  debug:
    type: bool
    value: true

  rate_limit:
    type: float64
    value: 99.5
```

### Supported Parameter Types

| Type | Example Value | Notes |
|------|--------------|-------|
| `string` | `"hello"` | String literals |
| `int` | `8080` | 64-bit integers |
| `float64` | `99.5` | Floating point numbers |
| `bool` | `true`, `false` | Boolean values |
| `time.Duration` | `"30s"`, `1000000000` | String format or nanoseconds |

### Duration Format

Duration parameters accept either:
- String literals: `"1h30m"`, `"500ms"`, `"2s"`
- Integer nanoseconds: `1000000000` (= 1 second)

### Parameter References

Reference parameters in constructor arguments using `%param_name%`:

```yaml
services:
  server:
    constructor:
      func: "server.New"
      args:
        - "%app_name%"
        - "%port%"
```

### Parameter Overrides

Later imports override earlier parameter values:

```yaml
# base.yaml
parameters:
  db_dsn:
    type: string
    value: "postgres://localhost/dev"

# production.yaml
imports:
  - ./base.yaml

parameters:
  db_dsn:
    type: string
    value: "postgres://prod-host/app"  # Overrides base.yaml
```

## Services

Services are objects constructed and managed by the container.

### Service Definition

```yaml
services:
  service_id:
    # Optional: Explicit type (inferred from constructor if omitted)
    type: "github.com/myapp.Service"

    # Constructor configuration
    constructor:
      # Function constructor
      func: "github.com/myapp.NewService"
      # OR method constructor
      method: "@other_service.CreateService"

      # Constructor arguments
      args:
        - "@dependency"           # Service reference
        - "%parameter%"           # Parameter reference
        - "!tagged:tag.name"      # Tagged services
        - "@.inner"               # Inner service (decorators only)
        - "literal string"        # Literal value
        - 123                     # Literal number
        - true                    # Literal bool

    # Lifecycle (default: shared=true)
    shared: true  # Singleton (cached)
    # shared: false creates new instance each time

    # Public API exposure
    public: true  # Generate public getter method

    # Service aliasing
    alias: "other_service"  # Alias to another service

    # Decoration
    decorates: "base_service"  # Decorate another service
    decoration_priority: 10    # Higher priority decorators wrap first

    # Tagging
    tags:
      - name: "tag.name"
        attribute1: "value1"
        priority: 100
```

### Service Lifecycle

**Shared Services (Singletons):**
```yaml
services:
  database:
    constructor:
      func: "db.New"
    shared: true  # Default
```

- Created once on first access
- Same instance returned on subsequent calls
- Thread-safe with mutex locking
- Suitable for: databases, HTTP clients, loggers

**Non-Shared Services (Factories):**
```yaml
services:
  request_context:
    constructor:
      func: "ctx.New"
    shared: false
```

- New instance created on each access
- No caching or locking overhead
- Suitable for: request handlers, temporary objects

### Service Aliases

Create multiple names for the same service:

```yaml
services:
  logger.impl:
    constructor:
      func: "log.New"

  logger:
    alias: "logger.impl"

  log:
    alias: "logger.impl"
```

### Public Services

Generate public getter methods for services:

```yaml
services:
  user_repo:
    constructor:
      func: "repo.NewUserRepository"
    public: true
```

Generated methods:
```go
func (c *Container) GetUserRepo() (*UserRepository, error)
func (c *Container) MustUserRepo() *UserRepository
```

### Method Constructors

Use service methods to construct other services:

```yaml
services:
  factory:
    constructor:
      func: "factory.New"
    shared: true

  processor:
    constructor:
      method: "@factory.CreateProcessor"
      args:
        - "@dependency"
    shared: false
```

Generated code:
```go
func (c *Container) buildProcessor() (*Processor, error) {
    factory, err := c.getFactory()
    if err != nil {
        return nil, err
    }
    dep, err := c.getDependency()
    if err != nil {
        return nil, err
    }
    return factory.CreateProcessor(dep), nil
}
```

### Generic Constructors

Support for Go generics with type arguments:

```yaml
services:
  events:
    constructor:
      func: "github.com/asp24/gendi/stdlib.NewChan[github.com/myapp/events.Event]"
      args:
        - 100  # buffer size
    public: true
```

Generated code:
```go
func (c *Container) buildEvents() (chan events.Event, error) {
    return stdlib.NewChan[events.Event](100), nil
}
```

### Service Decorators

Decorators wrap existing services to add behavior:

```yaml
services:
  logger:
    constructor:
      func: "log.New"

  logging_decorator:
    constructor:
      func: "log.NewDecorator"
      args:
        - "@.inner"  # Receives the decorated service
    decorates: logger
    decoration_priority: 10

  metrics_decorator:
    constructor:
      func: "metrics.NewDecorator"
      args:
        - "@.inner"
    decorates: logger
    decoration_priority: 20  # Higher priority wraps first
```

**Execution order:** `metrics(logging(logger))`

**Decorator Rules:**
- Must use `@.inner` in constructor args to receive the decorated service
- Multiple decorators on same service are ordered by `decoration_priority` (descending)
- Service ID of decorator becomes the alias after decoration
- Original service is renamed to `<decorator>.inner`

## Tags

Tags enable collecting multiple services that implement a common interface.

### Tag Definition

```yaml
tags:
  handler:
    # Optional: Element type inferred from usage if omitted
    element_type: "github.com/myapp.Handler"

    # Optional: Sort by tag attribute
    sort_by: "priority"

    # Optional: Generate public tag getter
    public: true

    # Optional: Auto-tag services implementing the interface
    autoconfigure: true
```

### Tag Attributes

- **`element_type`**: Go type of tagged services (required when `public: true` or `autoconfigure: true`)
- **`sort_by`**: Attribute name for sorting (incompatible with `autoconfigure`)
- **`public`**: Generate public getter for tagged collection
- **`autoconfigure`**: Automatically tag services implementing the interface type

### Tagged Services

```yaml
services:
  handler1:
    constructor:
      func: "github.com/myapp.NewHandler1"
    tags:
      - name: "handler"
        priority: 100

  handler2:
    constructor:
      func: "github.com/myapp.NewHandler2"
    tags:
      - name: "handler"
        priority: 200

  server:
    constructor:
      func: "github.com/myapp.NewServer"
      args:
        - "!tagged:handler"  # Receives []Handler sorted by priority
```

### Tag Sorting

When `sort_by` is specified, services are sorted by the tag attribute:

```yaml
tags:
  middleware:
    element_type: "github.com/myapp.Middleware"
    sort_by: "order"  # Sort by "order" attribute

services:
  auth:
    tags:
      - name: "middleware"
        order: 10

  logging:
    tags:
      - name: "middleware"
        order: 1

  metrics:
    tags:
      - name: "middleware"
        order: 5
```

Result: `[logging, metrics, auth]` (ascending order)

### Public Tag Getters

```yaml
tags:
  handler:
    element_type: "github.com/myapp.Handler"
    public: true
```

Generated method:
```go
func (c *Container) GetTaggedWithHandler() ([]Handler, error)
```

### Auto-Configuration

```yaml
tags:
  handler:
    element_type: "github.com/myapp.Handler"
    autoconfigure: true
```

All services whose constructor returns `Handler` interface are automatically tagged with `handler`.

**Rules:**
- `element_type` must be an interface type
- Cannot be combined with `sort_by`
- Services are tagged at IR build time

See `doc/spec/tags.md` for complete autoconfigure specification.

## Imports

Configuration files can import and override other configurations.

### Import Syntax

```yaml
imports:
  - ./base.yaml              # Relative path
  - ./services/*.yaml        # Glob pattern
  - ./**/gendi.yaml          # Recursive glob
  - github.com/pkg/stdlib    # Module import
```

### Import Resolution

- Relative paths resolved from importing file's directory
- Glob patterns expanded using doublestar matching
- Later imports override earlier ones
- Services with same ID are replaced completely

### Import Exclusions

Exclude specific files from glob pattern imports:

```yaml
imports:
  # Load all services except test files and internal files
  - path: ./services/*.yaml
    exclude:
      - ./services/test_*.yaml
      - ./services/internal/*.yaml

  # Glob in subdirectories with exclusions
  - path: ./config/**/*.yaml
    exclude:
      - ./config/**/dev_*.yaml
```

**Exclusion Features:**
- Supports full glob syntax (`*`, `?`, `[]`, `**`)
- Patterns resolved relative to importing file's directory
- Works with any import type (local, absolute, module-based)
- Exclusions take precedence over inclusions

### Import Merging

When merging configurations:

1. **Parameters**: Later values override earlier ones
2. **Tags**: Later definitions override earlier ones
3. **Services**: Later services completely replace earlier ones with same ID

### Module Imports

Import from Go modules:

```yaml
imports:
  - github.com/asp24/gendi/stdlib
```

Resolves to module's root directory and loads `gendi.yaml`.

### Best Practices

**Recommended structure:**

```
project/
├── gendi.yaml                 # Root (imports only)
├── services/
│   ├── database.yaml
│   ├── http.yaml
│   └── logging.yaml
└── config/
    ├── dev.yaml
    └── prod.yaml
```

**Root file:**
```yaml
imports:
  - ./services/*.yaml
  - ./config/dev.yaml
```

**Benefits:**
- Small, focused configuration files
- Easy to navigate and maintain
- Clear separation of concerns
- Environment-specific overrides

## Special Tokens

### The `$this` Token

`$this` is replaced with the Go package path of the config file's directory.

**Use in `type` field:**
```yaml
services:
  logger:
    type: "*$this.Logger"  # → *github.com/myapp/log.Logger
```

**Use in `func` field:**
```yaml
services:
  logger:
    constructor:
      func: "$this.NewLogger"  # → github.com/myapp/log.NewLogger
```

**Use in `method` field:**
```yaml
services:
  child:
    constructor:
      method: "@factory.CreateChild"  # → factory.CreateChild (no $this needed)
```

**Use in `!go:` arguments:**
```yaml
services:
  logger:
    constructor:
      func: "$this.NewLogger"
      args:
        - "!go:$this.DefaultLevel"  # → !go:github.com/myapp/log.DefaultLevel
```

**Use in `!field:!go:` arguments:**
```yaml
services:
  server:
    constructor:
      func: "$this.NewServer"
      args:
        - "!field:!go:$this.DefaultConfig.Host"  # → !field:!go:github.com/myapp.DefaultConfig.Host
```

**Benefits:**
- Eliminates repetitive package paths
- Makes configuration more portable
- Clearer intent (local vs external types)

### Service References

Reference other services using `@service_id`:

```yaml
services:
  database:
    constructor:
      func: "db.New"

  repository:
    constructor:
      func: "repo.New"
      args:
        - "@database"  # Injects database service
```

### Parameter References

Reference parameters using `%param_name%`:

```yaml
parameters:
  db_dsn:
    type: string
    value: "postgres://localhost/app"

services:
  database:
    constructor:
      func: "db.New"
      args:
        - "%db_dsn%"  # Injects parameter value
```

### Tagged References

Reference tagged service collections using `!tagged:tag_name`:

```yaml
services:
  server:
    constructor:
      func: "server.New"
      args:
        - "!tagged:handler"  # Injects []Handler
```

### Inner Service Reference

Reference the decorated service using `@.inner` (decorators only):

```yaml
services:
  decorator:
    constructor:
      func: "decorator.New"
      args:
        - "@.inner"  # The service being decorated
    decorates: base_service
```

### Spread Operator

Unpack slices into variadic parameters using `!spread:`:

```yaml
services:
  server:
    constructor:
      func: "server.New"  # func New(handlers ...Handler)
      args:
        - "!spread:!tagged:handler"  # Unpacks []Handler into ...Handler
```

**Spread with service reference:**
```yaml
services:
  handlers:
    constructor:
      func: "app.GetHandlers"  # Returns []Handler

  server:
    constructor:
      func: "server.New"  # func New(handlers ...Handler)
      args:
        - "!spread:@handlers"  # Unpacks handlers into ...Handler
```

**Spread Rules:**
- Only one `!spread:` per constructor call
- Must be the last argument
- Inner expression must resolve to a slice
- Target parameter must be variadic

See `doc/spec/services.md` for complete spread specification.

## Argument Syntax

Constructor arguments support multiple syntaxes:

| Syntax | Type | Example |
|--------|------|---------|
| `@service.id` | Service reference | `@database` |
| `@.inner` | Inner service | `@.inner` (decorators only) |
| `%param.name%` | Parameter | `%db_dsn%` |
| `!tagged:tag` | Tagged collection | `!tagged:handler` |
| `!spread:@service` | Spread service slice | `!spread:@handlers` |
| `!spread:!tagged:tag` | Spread tagged slice | `!spread:!tagged:middleware` |
| `!go:pkg.Symbol` | Go package-level var/const | `!go:os.Stdout` |
| `!field:@service.Field` | Service field access | `!field:@config.Host` |
| `!field:!go:pkg.Symbol.Field` | Go symbol field access | `!field:!go:http.DefaultClient.Timeout` |
| `@service.Method` | Method constructor | `@factory.Create` |
| `"string"` | String literal | `"localhost"` |
| `123` | Integer literal | `8080` |
| `45.6` | Float literal | `3.14` |
| `true`/`false` | Boolean literal | `true` |
| `null` | Null literal | `null` |

### Type Compatibility

gendi validates argument types at generation time:

- Service references must match parameter types
- Parameters are converted to target types
- Tagged collections must match slice element types
- Literals are type-checked against parameters

### Variadic Arguments

Variadic constructors accept multiple arguments:

```yaml
services:
  logger:
    constructor:
      func: "log.New"

  named_logger:
    constructor:
      method: "@logger.With"  # func With(args ...any) Logger
      args:
        - "channel"    # First variadic arg
        - "database"   # Second variadic arg
```

Generated code:
```go
return logger.With("channel", "database"), nil
```

## Complete Example

```yaml
# Import stdlib and base services
imports:
  - github.com/asp24/gendi/stdlib
  - ./services/base.yaml

# Configuration parameters
parameters:
  app_name:
    type: string
    value: "MyApp"

  db_dsn:
    type: string
    value: "postgres://localhost/myapp"

  http_timeout:
    type: time.Duration
    value: "30s"

# Tag definitions
tags:
  handler:
    element_type: "github.com/myapp.Handler"
    sort_by: "priority"
    public: true

# Service definitions
services:
  database:
    constructor:
      func: "github.com/myapp/db.New"
      args:
        - "%db_dsn%"
    shared: true

  user_repo:
    constructor:
      func: "github.com/myapp/repo.NewUserRepository"
      args:
        - "@database"
    shared: true
    public: true

  home_handler:
    constructor:
      func: "github.com/myapp/handlers.NewHome"
      args:
        - "@user_repo"
    tags:
      - name: "handler"
        priority: 10

  api_handler:
    constructor:
      func: "github.com/myapp/handlers.NewAPI"
      args:
        - "@user_repo"
    tags:
      - name: "handler"
        priority: 20

  http_server:
    constructor:
      func: "github.com/myapp/server.New"
      args:
        - "%app_name%"
        - "!spread:!tagged:handler"
    public: true

  # Decorator for logging
  logging_middleware:
    constructor:
      func: "github.com/myapp/middleware.NewLogging"
      args:
        - "@.inner"
        - "@stdlib.logger"
    decorates: http_server
    decoration_priority: 10
```

## See Also

- [Technical Specification](./spec/README.md)
- [Custom Compiler Passes](./custom-passes.md)
- [Standard Library Services](../stdlib/README.md)
- [Examples](../examples/)
