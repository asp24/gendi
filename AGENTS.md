This file provides guidance to LLM tools when working with code in this repository.

## Project Overview

**gendi** is a compile-time dependency injection container generator for Go. It reads YAML configuration files and generates type-safe, efficient container code with full compile-time validation—no runtime reflection.

Key characteristics:
- All dependencies are resolved and type-checked during code generation
- Generated code uses direct type assertions
- YAML-based declarative service definitions with imports and overrides
- Support for service lifecycle (shared/non-shared), decorators, tagged injection, and custom compiler passes

## Essential Commands

### Building and Testing
```bash
# Run all tests
go test ./...

# Build the CLI generator
go build ./cmd/gendi

# Run the generator manually
go run ./cmd/gendi --config=examples/basic/gendi.yaml --out=examples/basic/internal/di --pkg=di

# Regenerate all examples
just gen-examples
# OR
go generate ./...
```

### Running Examples
```bash
# Basic example
cd examples/basic && go generate && go run .

# Advanced example
cd examples/advanced && go generate && go run .

# Custom pass example
cd examples/custom-pass
go run ./tools/gendi --config=./cmd/gendi.yaml --out=./cmd --pkg=main
go run ./cmd
```

### Running a Single Test
```bash
# Run specific test
go test -run TestName ./path/to/package

# Example: run generator tests
go test -run TestGenerator ./generator

# Run with verbose output
go test -v -run TestName ./path/to/package
```

## Architecture

### Code Generation Pipeline

The generator follows a multi-stage pipeline:

1. **YAML Loading** (`yaml/` package)
   - Loads root YAML config with `yaml.LoadConfig()`
   - Resolves imports (supports glob patterns and relative paths)
   - Merges imported configs (later imports override earlier ones)
   - Resolves `$this` tokens to current package paths

2. **Compiler Passes** (`config.go`)
   - Optional transformation stage via `di.ApplyPasses()`
   - Passes implement `Pass` interface with `Process(*Config) (*Config, error)`
   - Used for custom conventions, auto-tagging, argument modification
   - See `examples/custom-pass` for practical implementation

3. **Type Resolution** (`typeres/` package)
   - Uses `golang.org/x/tools/go/packages` to load Go types
   - Resolves type strings to `go/types.Type`
   - Handles generic constructors with type arguments
   - Validates constructor signatures

4. **IR Building** (`ir/` package)
   - Multi-phase transformation from `di.Config` to IR `Container`
   - **Phase 1**: Build parameters, tags, and services
   - **Phase 2**: Resolve constructors, decorators, and dependencies
   - **Phase 3**: Validate (circular dependencies, type compatibility)
   - **Phase 4**: Optimize (prune unreachable services, optimize shared flags)
   - IR represents fully analyzed and validated dependency graph

5. **Code Generation** (`generator/` package)
   - Converts IR to Go source code
   - Renders: parameters map, container struct, getter methods
   - Manages imports with `ImportManager`
   - Handles identifier collision with `IdentGenerator`
   - Formats output with `go/format`

### Key Package Roles

- **`di` (root package)**: Core config types (`Config`, `Service`, `Parameter`, `Tag`) and `Pass` interface
- **`cmd/`**: CLI implementation with flag parsing and orchestration
- **`yaml/`**: YAML parsing, import resolution, `$this` token replacement
- **`ir/`**: Intermediate representation and multi-phase analysis
- **`generator/`**: Code generation from IR to Go source
- **`typeres/`**: Wrapper around `go/packages` for type resolution
- **`parameters/`**: Runtime parameter provider interface and implementations
- **`stdlib/`**: Pre-built factory functions for stdlib types (channels, HTTP clients, loggers)

### Important Files

- `config.go`: Core config structures and `Pass` interface
- `ir/builder.go`: IR construction orchestration
- `generator/generator.go`: Main generator entry point
- `cmd/cli.go`: CLI workflow (`Run()` and `Generate()`)

## Configuration Concepts

### Service References and Arguments

Constructor arguments use special syntax:
- `@service.id` - Reference to another service
- `@.inner` - Inner service (decorators only)
- `%param.name%` - Parameter reference
- `!tagged:tag.name` - Tagged services collection
- `!spread:@service` - Spread slice into variadic parameters
- `!spread:!tagged:tag` - Spread tagged collection into variadic parameters
- `!go:pkg.Var` - Go package-level variable or constant (e.g., `!go:os.Stdout`, `!go:log.LstdFlags`)
- `!field:@service.Field` - Field access on a service (e.g., `!field:@config.Host`, `!field:@config.Database.DSN`)
- `!field:!go:pkg.Symbol.Field` - Field access on a Go package-level variable (e.g., `!field:!go:http.DefaultClient.Timeout`)
- `@service.Method` - Method constructor
- `literal` - YAML scalar literal

### Service Lifecycle

- **Shared (singleton)**: Created once on first access, cached, thread-safe
- **Non-shared (factory)**: New instance on each access, no caching

### Decorators

Decorators wrap existing services using `@.inner` reference. Multiple decorators are ordered by `decoration_priority` (higher priority wraps first).

### Tagged Injection

Services can be tagged and collected as `[]ElementType`. Tags support:
- Optional `element_type` (inferred from usage if omitted, required when public)
- Sorting by attribute (e.g., `sort_by: "priority"`)
- Public getters when `public: true`

### Imports

Configuration files can import others:
- Relative paths resolved from importing file
- Glob patterns supported (`./services/*.yaml`)
- Later imports override earlier ones
- Each file has its own `$this` context

#### Import Exclusions

Exclude files from imports using glob patterns or directory paths:

```yaml
imports:
  # Load all services except test files and an entire directory
  - path: ./services/**/*.yaml
    exclude:
      - ./services/test_*.yaml
      - ./services/internal

  # Glob in subdirectories with exclusions
  - path: ./config/**/*.yaml
    exclude:
      - ./config/**/dev_*.yaml
```

Features:
- Exclusion entries can be glob patterns (`*`, `?`, `[]`, `**`) or directory paths
- Directory paths exclude all files under that directory
- Patterns resolved relative to importing file's directory
- Works with any import type (local, absolute, module-based)
- Exclusions take precedence over inclusions
- Backward compatible - `exclude` field is optional

### The `$this` Token

`$this` is replaced with the Go package path of the config file's directory:
- In `type` field: `*$this.Logger` → `*github.com/user/app.Logger`
- In `func` field: `$this.NewLogger` → `github.com/user/app.NewLogger`
- In `!go:` args: `!go:$this.DefaultLevel` → `!go:github.com/user/app.DefaultLevel`
- In `!field:!go:` args: `!field:!go:$this.DefaultConfig.Host` → `!field:!go:github.com/user/app.DefaultConfig.Host`
- Eliminates repetitive package paths

## Testing Strategy

Tests use table-driven approach with golden files:
- `generator/generator_test.go`: Golden file comparisons for generated output
- `ir/*_test.go`: IR phase validation and transformation tests
- `yaml/*_test.go`: Config loading and import resolution tests

When updating generator behavior:
1. Run tests to see failures
2. Review generated output carefully
3. Update golden files if changes are correct
4. Regenerate examples with `just gen-examples`

## Commit Style

This project uses short, imperative, unscoped commit messages:
- ✅ "Fix circular dependency detection"
- ✅ "Add support for variadic constructors"
- ✅ "Regenerate examples"
- ❌ "feat(ir): add circular dependency detection"

## Key Design Principles

1. **Fail at generation time, not runtime**: All type errors, missing dependencies, and circular references are caught during code generation
2. **No runtime reflection**: Generated code uses direct type assertions and function calls
3. **Explicit over implicit**: No autowiring—all dependencies must be explicitly configured
4. **Type safety**: Constructor signatures are strictly validated, no `any` or `interface{}` service types
5. **Deterministic output**: Generated code is consistent and reproducible

## Custom Compiler Passes

For project-specific conventions, create custom passes:

```go
type MyPass struct{}

func (p *MyPass) Name() string { return "my-pass" }

func (p *MyPass) Process(cfg *di.Config) (*di.Config, error) {
    // Mutate cfg (add services, modify args, etc.)
    return cfg, nil
}
```

Use in custom generator:
```go
// tools/gendi/main.go
func main() {
    passes := []di.Pass{&MyPass{}}
    cmd.Run(flag.CommandLine, passes)
}
```

See `examples/custom-pass` for complete example.

## Common Patterns

### Reading Configuration
```go
cfg, err := yaml.LoadConfig("gendi.yaml")
cfg, err = di.ApplyPasses(cfg, passes)
```

### Generating Container
```go
opts := pipeline.Options{
    Out:     "./internal/di",
    Package: "di",
}
opts.Finalize()

code, err := pipeline.Generate(cfg, opts)
```

### Generated Container Usage
```go
container := di.NewContainer(nil) // nil uses default parameters
service, err := container.GetServiceName()
```

## Generated File Conventions

- Generated files follow `*_gen.go` naming
- All contain banner: `// Code generated by gendi; DO NOT EDIT.`
- Never edit generated files manually—modify YAML config or generator instead
