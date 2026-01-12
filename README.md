# gendi - Compile-Time Dependency Injection for Go

`gendi` is a compile-time dependency injection container generator for Go. It reads YAML configuration files and generates type-safe, efficient container code with full compile-time validation.

## Features

- **Compile-time type safety** - All dependencies resolved and type-checked during code generation
- **Zero runtime reflection** - Generated code uses direct type assertions
- **YAML configuration** - Declarative service definitions with imports and overrides
- **Service lifecycle** - Shared (singleton) and non-shared (factory) services
- **Tagged injection** - Collect multiple services by tag with custom sorting
- **Service decoration** - Decorator pattern with priority ordering
- **Method constructors** - Use service methods as constructors
- **Variadic functions** - Full support for variadic constructors
- **Generic constructors** - Support for Go generics with type arguments
- **Custom compiler passes** - Transform configuration before generation
- **Parameter injection** - Type-safe parameter references with automatic conversion
- **Circular dependency detection** - Catches circular references at generation time
- **Public API generation** - Expose selected services via public getter methods
- **Standard library factories** - Ready-to-use factories for common stdlib types

## Installation

Add gendi as a tool dependency to your project:

```bash
go get -tool github.com/asp24/gendi/cmd/gendi
```

This adds gendi to your `go.mod` and allows running it via `go tool gendi`.

## Quick Start

### 1. Create a service configuration

**gendi.yaml:**
```yaml
parameters:
  db_dsn:
    type: string
    value: "postgres://localhost/myapp"

services:
  database:
    constructor:
      func: "github.com/myapp/db.New"
      args:
        - "%db_dsn%"  # Parameter reference
    shared: true

  user_repo:
    constructor:
      func: "github.com/myapp/repo.NewUserRepository"
      args:
        - "@database"  # Service reference
    shared: true
    public: true  # Exposed via public getter
```

### 2. Generate the container

```bash
go tool gendi --config=gendi.yaml --out=./di --pkg=di
```

Or use `go:generate`:
```go
//go:generate go tool gendi --config=gendi.yaml --out=./di --pkg=di
```

### 3. Use the generated container

```go
package main

import "github.com/myapp/di"

func main() {
    container := di.NewContainer(nil)

    userRepo, err := container.GetUserRepo()
    if err != nil {
        panic(err)
    }

    // Use userRepo...
}
```

## Configuration Reference

### Parameters

Parameters are typed configuration values injected using `%name%` syntax:

```yaml
parameters:
  app_name:
    type: string
    value: "MyApp"

  port:
    type: int
    value: 8080

  timeout:
    type: duration
    value: "30s"

  debug:
    type: bool
    value: true
```

**Supported types:** `string`, `int`, `float`, `bool`, `duration`

### Services

Services are objects constructed and managed by the container:

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
    # shared: false would create new instance each time

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

### Tags

Tags enable collecting multiple services that implement a common interface:

```yaml
tags:
  # Optional: Element type inferred from usage if omitted (required when public)
  handler:
    element_type: "github.com/myapp.Handler"
    sort_by: "priority"  # Sort by tag attribute
    public: true         # Generate public tag getter

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

# Generated public tag getter:
# func (c *Container) GetTaggedWithHandler() ([]Handler, error)
```

### Imports

Configuration files can import and override other configurations:

```yaml
imports:
  - ./base.yaml           # Relative imports
  - ./services/*.yaml     # Glob patterns
  - ./**/gendi.yaml       # Recursive glob

parameters:
  # Override imported parameters
  db_dsn:
    type: string
    value: "postgres://prod-host/myapp"
```

**Import resolution:**
- Later imports override earlier ones
- Parameters, tags, and services are merged
- Services with the same ID are replaced

### Special Tokens

- `$this` - Replaced with current file's package path
- `@service` - Service reference
- `%parameter%` - Parameter reference
- `!tagged:tag` - Tagged services collection
- `!spread:@service` - Spread slice into variadic parameters
- `!spread:!tagged:tag` - Spread tagged collection into variadic parameters
- `@.inner` - Inner service (decorators only)
- `@service.Method` - Method constructor

## CLI Usage

```bash
go tool gendi [flags]

Flags:
  --config string      Root YAML configuration file (required)
  --out string         Output directory or file (required)
  --pkg string         Go package name (required)
  --container string   Container struct name (default: "Container")
  --build-tags string  Build tags for generated file
  --verbose           Enable verbose logging
```

**Examples:**

```bash
# Generate to directory
go tool gendi --config=gendi.yaml --out=./di --pkg=di

# Generate specific file
go tool gendi --config=gendi.yaml --out=./di/container_gen.go --pkg=di

# With build tags
go tool gendi --config=gendi.yaml --out=./di --pkg=di --build-tags=integration
```

## Custom Compiler Passes

Compiler passes transform configuration before code generation, enabling project-specific conventions and patterns.

### Implementing a Pass

```go
package passes

import di "github.com/asp24/gendi"

type AutoTagPass struct{}

func (p *AutoTagPass) Name() string {
    return "auto-tag"
}

func (p *AutoTagPass) Process(cfg *di.Config) (*di.Config, error) {
    for id, svc := range cfg.Services {
        // Auto-tag services by naming convention
        if strings.HasSuffix(id, ".handler") {
            svc.Tags = append(svc.Tags, di.ServiceTag{
                Name: "http.handler",
            })
            cfg.Services[id] = svc
        }
    }
    return cfg, nil
}
```

### Creating a Custom Generator

**tools/gendi/main.go:**
```go
package main

import (
    "flag"
    "fmt"
    "os"

    "github.com/asp24/gendi"
    "github.com/asp24/gendi/cmd"
    "github.com/myapp/internal/passes"
)

func main() {
    customPasses := []gendi.Pass{
        &passes.AutoTagPass{},
        &passes.ValidationPass{},
    }

    if err := cmd.Run(flag.CommandLine, customPasses); err != nil {
        fmt.Fprintf(os.Stderr, "%v\n", err)
        os.Exit(1)
    }
}
```

**Usage:**
```bash
go run ./tools/gendi --config=gendi.yaml --out=./di --pkg=di
```

See [examples/custom-pass](./examples/custom-pass) for a complete example with:
- Auto-tagging by naming convention
- Channel-specific structured logging
- Variadic method constructors

## Key Concepts

### Service Lifecycle

**Shared (Singleton):**
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

**Non-Shared (Factory):**
```yaml
services:
  request_context:
    constructor:
      func: "ctx.New"
    shared: false
```
- New instance created on each access
- No caching or locking overhead

### Service Decoration

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

### Variadic Functions

Full support for variadic constructors:

```yaml
services:
  logger:
    constructor:
      func: "log.New"

  named_logger:
    constructor:
      method: "@logger.With"
      args:
        - "channel"    # First variadic arg
        - "database"   # Second variadic arg
```

Generated code:
```go
return logger.With("channel", "database"), nil
```

**Spread Operator:**

Use `!spread:` to unpack slices into variadic parameters:

```yaml
services:
  handler.a:
    constructor:
      func: "app.NewHandlerA"
    tags:
      - name: handler

  handler.b:
    constructor:
      func: "app.NewHandlerB"
    tags:
      - name: handler

  server:
    constructor:
      func: "app.NewServer"  # NewServer(handlers ...Handler)
      args:
        - "!spread:!tagged:handler"  # Unpacks []Handler into ...Handler
    public: true
```

Generated code:
```go
func (c *Container) buildServer() (*Server, error) {
    tagged_0, err := c.getTaggedWithHandler()
    if err != nil {
        return nil, err
    }
    return app.NewServer(tagged_0...), nil
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

## Standard Library Factories

The `stdlib` package provides ready-to-use factory functions for common Go stdlib types.

### Installation

Import the stdlib services in your configuration:

```yaml
imports:
  - github.com/asp24/gendi/stdlib
```

### Available Factories

| Package | Function | Description |
|---------|----------|-------------|
| `stdlib` | `NewChan[T](size int)` | Generic buffered channel |
| `stdlib` | `NewHTTPClient(timeout)` | HTTP client with timeout |
| `stdlib` | `NewHTTPClientWithTransport(timeout, transport)` | HTTP client with custom transport |
| `stdlib` | `NewHTTPTransport(maxIdle, maxIdlePerHost, idleTimeout)` | HTTP transport with pooling |
| `stdlib` | `NewSlogTextHandler(writer, level)` | Text log handler |
| `stdlib` | `NewSlogJSONHandler(writer, level)` | JSON log handler |
| `stdlib` | `NewSlogLogger(handler)` | Structured logger |
| `stdlib` | `NewStdout()` | Standard output writer |
| `stdlib` | `NewStderr()` | Standard error writer |

### Pre-configured Services

The stdlib gendi.yaml provides ready-to-use services:

```yaml
imports:
  - github.com/asp24/gendi/stdlib

services:
  my_service:
    constructor:
      func: "github.com/myapp.NewService"
      args:
        - "@stdlib.http.client"  # Pre-configured HTTP client
        - "@stdlib.logger"       # Pre-configured logger
```

**Available services:**
- `stdlib.http.client` - HTTP client with 30s timeout
- `stdlib.http.transport` - HTTP transport with connection pooling
- `stdlib.http.client_with_transport` - HTTP client with custom transport
- `stdlib.stdout`, `stdlib.stderr` - I/O writers
- `stdlib.slog.handler.text`, `stdlib.slog.handler.json` - Log handlers
- `stdlib.slog` - Structured logger (uses text handler by default)
- `stdlib.logger` - Alias to `stdlib.slog`

**Configurable parameters:**
- `stdlib.http.timeout` (default: 30s)
- `stdlib.http.max_idle_conns` (default: 100)
- `stdlib.http.max_idle_conns_per_host` (default: 10)
- `stdlib.http.idle_conn_timeout` (default: 90s)
- `stdlib.slog.level` (default: Info)

### Custom Channel Types

For application-specific channel types, use the generic `NewChan`:

```yaml
services:
  order_events:
    constructor:
      func: "github.com/asp24/gendi/stdlib.NewChan[github.com/myapp/orders.OrderEvent]"
      args: [100]
    public: true
```

## Examples

The repository includes three complete examples:

### [Basic Example](./examples/basic)
Demonstrates core features:
- Parameters and service injection
- Tagged injection with sorting
- Service decorators
- Method-based constructors

```bash
cd examples/basic
go generate
go run .
```

### [Advanced Example](./examples/advanced)
Shows advanced patterns:
- Configuration imports and overrides
- Decorator chains with priorities
- Tagged injection with interfaces
- Multi-environment configuration

```bash
cd examples/advanced
go generate
go run .
```

### [Custom Pass Example](./examples/custom-pass)
Production-ready custom compiler passes:
- Auto-tagging by naming convention
- Channel-specific structured logging
- Custom generator binary
- Variadic method constructors

```bash
cd examples/custom-pass
go run ./tools/gendi --config=./cmd/gendi.yaml --out=./cmd --pkg=main
go run ./cmd
```

## API Documentation

### Core Packages

- **`github.com/asp24/gendi`** - Configuration types and Pass interface
- **`github.com/asp24/gendi/yaml`** - YAML config loader with imports
- **`github.com/asp24/gendi/generator`** - Container code generator
- **`github.com/asp24/gendi/cmd`** - Reusable CLI for custom generators
- **`github.com/asp24/gendi/parameters`** - Runtime parameter provider
- **`github.com/asp24/gendi/stdlib`** - Factory functions for stdlib types
- **`github.com/asp24/gendi/ir`** - Intermediate representation

### Using as a Library

```go
package main

import (
    "flag"
    "fmt"
    "os"

    "github.com/asp24/gendi"
    "github.com/asp24/gendi/cmd"
)

func main() {
    // Define custom compiler passes (optional)
    customPasses := []gendi.Pass{
        &MyPass{},
    }

    // Run gendi with custom passes
    if err := cmd.Run(flag.CommandLine, customPasses); err != nil {
        fmt.Fprintf(os.Stderr, "%v\n", err)
        os.Exit(1)
    }
}
```

This is the same pattern used in [examples/custom-pass/tools/gendi/main.go](./examples/custom-pass/tools/gendi/main.go).

For programmatic usage without CLI, use the lower-level API:

```go
import (
    "github.com/asp24/gendi"
    "github.com/asp24/gendi/generator"
    "github.com/asp24/gendi/yaml"
)

// Load and apply passes
cfg, _ := yaml.LoadConfig("gendi.yaml")
cfg, _ = gendi.ApplyPasses(cfg, customPasses)

// Generate
opts := generator.Options{Out: "./di", Package: "di"}
opts.Finalize()
gen := generator.New(cfg, opts)
code, _ := gen.Generate()
```

## Generated Container API

The generated container provides:

**Constructor:**
```go
func NewContainer(params parameters.Provider) *Container
```

**Public Service Getters:**
```go
func (c *Container) GetServiceName() (ServiceType, error)
```

**Default Parameters:**
```go
var DefaultContainerParameters = parameters.NewProviderMap(map[string]any{
    "param_name": "value",
})
```

**Thread Safety:**
- All service getters are thread-safe
- Shared services use mutex locking
- Non-shared services have no locking overhead

## Requirements

- Go 1.21 or later
- No runtime dependencies for generated code (except `github.com/asp24/gendi/parameters`)

## License

[Add your license here]

## Contributing

[Add contributing guidelines here]

## Related Projects

- [google/wire](https://github.com/google/wire) - Compile-time dependency injection with code generation
- [uber-go/fx](https://github.com/uber-go/fx) - Runtime dependency injection framework
- [Symfony DependencyInjection](https://symfony.com/doc/current/components/dependency_injection.html) - Inspiration for YAML configuration format
