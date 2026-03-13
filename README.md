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
# yaml-language-server: $schema=https://raw.githubusercontent.com/asp24/gendi/master/gendi.schema.json

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

> **💡 Tip:** Add the schema comment at the top of your YAML files to get autocomplete and validation in editors that support YAML schemas (VS Code, IntelliJ, etc.)

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

## Core Concepts

### Parameters

Typed configuration values injected using `%name%` syntax. Supported types: `string`, `int`, `float64`, `bool`, `time.Duration`.

### Services

Objects constructed and managed by the container. Services can be:
- **Shared (singleton)**: Created once, cached, thread-safe
- **Non-shared (factory)**: New instance on each access
- **Public**: Exposed via public getter methods
- **Decorated**: Wrapped by decorator services

### Tags

Collect multiple services implementing a common interface. Tags support:
- Custom sorting by attributes
- Auto-configuration (automatic tagging)
- Public getters for tagged collections

### Imports

Configuration files can import and override other configurations using relative paths, glob patterns, or module imports.

**📖 See [Configuration Reference](./doc/configuration.md) for complete YAML syntax and examples.**

## CLI Usage

```bash
go tool gendi [flags]

Flags:
  --config string      Root YAML configuration file (required)
  --out string         Output directory or file (required)
  --pkg string         Go package name (required)
  --container string   Container struct name (default: "Container")
  --strict            Enable strict validation (default: true)
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

Compiler passes transform configuration before code generation, enabling project-specific conventions:

```go
type AutoTagPass struct{}

func (p *AutoTagPass) Name() string { return "auto-tag" }

func (p *AutoTagPass) Process(cfg *di.Config) (*di.Config, error) {
    for id, svc := range cfg.Services {
        if strings.HasSuffix(id, ".handler") {
            svc.Tags = append(svc.Tags, di.ServiceTag{Name: "http.handler"})
            cfg.Services[id] = svc
        }
    }
    return cfg, nil
}
```

**📖 See [Custom Passes Guide](./doc/custom-passes.md) for complete documentation and examples.**

## Standard Library Services

Pre-configured services for common stdlib types (HTTP clients, loggers, channels):

```yaml
imports:
  - github.com/asp24/gendi/stdlib/gendi.yaml

services:
  my_service:
    constructor:
      func: "github.com/myapp.NewService"
      args:
        - "@stdlib.http.client"  # Pre-configured HTTP client
        - "@stdlib.logger"       # Pre-configured slog logger
```

**📖 See [stdlib/README.md](./stdlib/README.md) for all available services and factory functions.**

## Examples

| Example | Description | Run |
|---------|-------------|-----|
| [basic](./examples/basic) | Parameters, tagged injection, decorators | `cd examples/basic && go generate && go run .` |
| [advanced](./examples/advanced) | Imports, overrides, decorator chains | `cd examples/advanced && go generate && go run .` |
| [custom-pass](./examples/custom-pass) | Custom compiler passes, auto-tagging | `cd examples/custom-pass && go run ./tools/gendi --config=./cmd/gendi.yaml --out=./cmd --pkg=main && go run ./cmd` |
| [spread](./examples/spread) | Spread operator for variadic functions | `cd examples/spread && go generate && go run .` |
| [decorator](./examples/decorator) | Service decoration patterns | `cd examples/decorator && go generate && go run .` |

## Documentation

- **[Configuration Reference](./doc/configuration.md)** - Complete YAML syntax and examples
- **[Custom Passes Guide](./doc/custom-passes.md)** - Writing custom compiler passes
- **[stdlib Services](./stdlib/README.md)** - Pre-configured standard library services
- **[Technical Specification](./doc/spec/README.md)** - Architecture and design decisions
- **[API Documentation](https://pkg.go.dev/github.com/asp24/gendi)** - Go package documentation

## Requirements

- Go 1.25.4 or later
- No runtime dependencies for generated code (except `github.com/asp24/gendi/parameters`)

## Related Projects

- [google/wire](https://github.com/google/wire) - Compile-time DI with code generation
- [uber-go/fx](https://github.com/uber-go/fx) - Runtime dependency injection framework
- [Symfony DependencyInjection](https://symfony.com/doc/current/components/dependency_injection.html) - Inspiration for YAML format
