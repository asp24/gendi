# Custom Compiler Passes

Compiler passes transform configuration before code generation, enabling project-specific conventions and patterns.

## Table of Contents

- [Overview](#overview)
- [Pass Interface](#pass-interface)
- [Creating a Pass](#creating-a-pass)
- [Building a Custom Generator](#building-a-custom-generator)
- [Common Use Cases](#common-use-cases)
- [Best Practices](#best-practices)
- [Complete Example](#complete-example)

## Overview

Compiler passes allow you to programmatically modify the DI configuration before container generation. This enables:

- **Auto-tagging** by naming conventions
- **Service registration** from code analysis
- **Argument transformation** (e.g., adding logging)
- **Validation** of custom constraints
- **Configuration normalization**

### When to Use Passes

Use compiler passes when you need:

- Project-specific conventions (e.g., auto-tag all `*Handler` services)
- Dynamic service registration from external sources
- Custom validation rules beyond type checking
- Configuration preprocessing (e.g., environment variable expansion)
- Code generation from annotations

### When NOT to Use Passes

Don't use passes for:

- Simple configuration changes (use YAML imports/overrides instead)
- Type transformations (use decorators instead)
- Runtime behavior modification (implement in your services)

## Pass Interface

All compiler passes implement the `Pass` interface:

```go
package di

type Pass interface {
    Name() string
    Process(cfg *Config) (*Config, error)
}
```

### Interface Methods

**`Name() string`**
- Returns a unique identifier for the pass
- Used in error messages and logging
- Should be lowercase with hyphens (e.g., `"auto-tag"`)

**`Process(cfg *Config) (*Config, error)`**
- Receives the current configuration
- Returns the transformed configuration
- Returns an error if transformation fails
- **Must not modify the input config** (create a copy if needed)

## Creating a Pass

### Basic Pass Structure

```go
package passes

import (
    di "github.com/asp24/gendi"
)

type AutoTagPass struct {
    // Optional: configuration fields
}

func (p *AutoTagPass) Name() string {
    return "auto-tag"
}

func (p *AutoTagPass) Process(cfg *di.Config) (*di.Config, error) {
    // Transform cfg
    // Return modified config or error
    return cfg, nil
}
```

### Modifying the Configuration

The `Config` struct contains three maps:

```go
type Config struct {
    Parameters map[string]Parameter
    Tags       map[string]Tag
    Services   map[string]Service
}
```

**Safe modification pattern:**

```go
func (p *MyPass) Process(cfg *di.Config) (*di.Config, error) {
    // Iterate over services
    for id, svc := range cfg.Services {
        // Modify service
        svc.Tags = append(svc.Tags, di.ServiceTag{
            Name: "my-tag",
        })

        // Write back to map (services are value types)
        cfg.Services[id] = svc
    }

    return cfg, nil
}
```

**Important:** Services, parameters, and tags are value types. Always write back to the map after modification.

### Adding Services

```go
func (p *MyPass) Process(cfg *di.Config) (*di.Config, error) {
    cfg.Services["new_service"] = di.Service{
        Constructor: &di.Constructor{
            Func: "github.com/myapp.NewService",
            Args: []di.Argument{
                {Kind: di.ArgLiteral, Literal: di.NewStringLiteral("value")},
            },
        },
        Shared: true,
    }

    return cfg, nil
}
```

### Adding Tags

```go
func (p *MyPass) Process(cfg *di.Config) (*di.Config, error) {
    cfg.Tags["my-tag"] = di.Tag{
        ElementType: "github.com/myapp.Handler",
        SortBy:      "priority",
    }

    return cfg, nil
}
```

### Adding Parameters

```go
func (p *MyPass) Process(cfg *di.Config) (*di.Config, error) {
    cfg.Parameters["new_param"] = di.Parameter{
        Type:  "string",
        Value: "default-value",
    }

    return cfg, nil
}
```

## Building a Custom Generator

To use custom passes, create a custom generator binary:

### 1. Create Generator Package

**tools/gendi/main.go:**
```go
package main

import (
    "flag"
    "fmt"
    "os"

    di "github.com/asp24/gendi"
    "github.com/asp24/gendi/cmd"
    "github.com/myapp/internal/passes"
)

func main() {
    // Define custom compiler passes
    customPasses := []di.Pass{
        &passes.AutoTagPass{},
        &passes.ValidationPass{},
    }

    // Run gendi with custom passes
    if err := cmd.Run(flag.CommandLine, customPasses); err != nil {
        fmt.Fprintf(os.Stderr, "%v\n", err)
        os.Exit(1)
    }
}
```

### 2. Run Custom Generator

```bash
# Build custom generator
go build -o bin/gendi ./tools/gendi

# Run custom generator
./bin/gendi --config=gendi.yaml --out=./di --pkg=di

# Or use go run
go run ./tools/gendi --config=gendi.yaml --out=./di --pkg=di
```

### 3. Integrate with go:generate

```go
//go:generate go run ./tools/gendi --config=gendi.yaml --out=./di --pkg=di
```

## Common Use Cases

### Auto-Tagging by Convention

Tag services based on naming patterns:

```go
type AutoTagPass struct{}

func (p *AutoTagPass) Name() string {
    return "auto-tag"
}

func (p *AutoTagPass) Process(cfg *di.Config) (*di.Config, error) {
    for id, svc := range cfg.Services {
        // Tag all services ending with ".handler"
        if strings.HasSuffix(id, ".handler") {
            svc.Tags = append(svc.Tags, di.ServiceTag{
                Name: "http.handler",
            })
            cfg.Services[id] = svc
        }

        // Tag all services starting with "middleware."
        if strings.HasPrefix(id, "middleware.") {
            svc.Tags = append(svc.Tags, di.ServiceTag{
                Name: "http.middleware",
            })
            cfg.Services[id] = svc
        }
    }

    return cfg, nil
}
```

### Adding Logging to Services

Automatically inject logger into all services:

```go
type LoggingPass struct{}

func (p *LoggingPass) Name() string {
    return "auto-logging"
}

func (p *LoggingPass) Process(cfg *di.Config) (*di.Config, error) {
    for id, svc := range cfg.Services {
        // Skip logger itself
        if id == "logger" {
            continue
        }

        // Add logger as first argument
        if svc.Constructor != nil {
            svc.Constructor.Args = append(
                []di.Argument{{Kind: di.ArgServiceRef, Value: "logger"}},
                svc.Constructor.Args...,
            )
            cfg.Services[id] = svc
        }
    }

    return cfg, nil
}
```

### Custom Validation

Validate custom business rules:

```go
type ValidationPass struct{}

func (p *ValidationPass) Name() string {
    return "validation"
}

func (p *ValidationPass) Process(cfg *di.Config) (*di.Config, error) {
    // Ensure all public services are shared
    for id, svc := range cfg.Services {
        if svc.Public && !svc.Shared {
            return nil, fmt.Errorf(
                "service %q is public but not shared", id,
            )
        }
    }

    // Ensure required services exist
    requiredServices := []string{"logger", "database"}
    for _, required := range requiredServices {
        if _, exists := cfg.Services[required]; !exists {
            return nil, fmt.Errorf(
                "required service %q not found", required,
            )
        }
    }

    return cfg, nil
}
```

### Priority-Based Ordering

Automatically assign priorities based on registration order:

```go
type PriorityPass struct{}

func (p *PriorityPass) Name() string {
    return "auto-priority"
}

func (p *PriorityPass) Process(cfg *di.Config) (*di.Config, error) {
    priority := 1000

    for id, svc := range cfg.Services {
        // Add priority to all handler tags
        for i, tag := range svc.Tags {
            if tag.Name == "handler" && tag.Attributes["priority"] == nil {
                if svc.Tags[i].Attributes == nil {
                    svc.Tags[i].Attributes = make(map[string]interface{})
                }
                svc.Tags[i].Attributes["priority"] = priority
                priority += 10
            }
        }

        cfg.Services[id] = svc
    }

    return cfg, nil
}
```

### Environment-Based Configuration

Load configuration from environment:

```go
type EnvPass struct{}

func (p *EnvPass) Name() string {
    return "env-config"
}

func (p *EnvPass) Process(cfg *di.Config) (*di.Config, error) {
    // Override parameters from environment
    for name, param := range cfg.Parameters {
        envKey := "APP_" + strings.ToUpper(
            strings.ReplaceAll(name, ".", "_"),
        )

        if envValue := os.Getenv(envKey); envValue != "" {
            param.Value = envValue
            cfg.Parameters[name] = param
        }
    }

    return cfg, nil
}
```

## Best Practices

### 1. Keep Passes Focused

Each pass should do one thing well:

```go
// ✅ Good: Focused pass
type AutoTagHandlerPass struct{}

// ❌ Bad: Does too much
type AutoConfigureEverythingPass struct{}
```

### 2. Document Pass Behavior

Add clear documentation:

```go
// AutoTagPass automatically tags services by naming convention:
// - Services ending with ".handler" → "http.handler" tag
// - Services starting with "middleware." → "http.middleware" tag
type AutoTagPass struct{}
```

### 3. Fail Fast with Clear Errors

Return descriptive errors:

```go
if svc.Constructor == nil {
    return nil, fmt.Errorf(
        "service %q: missing constructor (required by auto-tag pass)",
        id,
    )
}
```

### 4. Don't Assume Configuration State

Validate your assumptions:

```go
func (p *MyPass) Process(cfg *di.Config) (*di.Config, error) {
    if cfg.Services == nil {
        cfg.Services = make(map[string]di.Service)
    }

    // Now safe to modify cfg.Services
    // ...
}
```

### 5. Use Source Locations

Add source location tracking for better error messages:

```go
svc.Constructor.SourceLoc = &srcloc.Location{
    File: "auto-generated",
    Line: 0,
}
```

### 6. Test Your Passes

Write unit tests:

```go
func TestAutoTagPass(t *testing.T) {
    cfg := &di.Config{
        Services: map[string]di.Service{
            "home.handler": {
                Constructor: &di.Constructor{
                    Func: "app.NewHomeHandler",
                },
            },
        },
    }

    pass := &AutoTagPass{}
    result, err := pass.Process(cfg)

    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    svc := result.Services["home.handler"]
    if len(svc.Tags) != 1 || svc.Tags[0].Name != "http.handler" {
        t.Errorf("expected http.handler tag, got: %v", svc.Tags)
    }
}
```

### 7. Order Matters

Passes run in sequence. Be aware of dependencies:

```go
customPasses := []di.Pass{
    &NormalizationPass{},  // Run first: normalize config
    &AutoTagPass{},        // Then: add tags
    &ValidationPass{},     // Finally: validate result
}
```

## Complete Example

See [examples/custom-pass](../examples/custom-pass) for a production-ready implementation featuring:

### Custom Pass: Channel Logger

Automatically adds structured logging with channel names to method constructors:

```go
type ChannelLoggerPass struct{}

func (p *ChannelLoggerPass) Name() string {
    return "channel-logger"
}

func (p *ChannelLoggerPass) Process(cfg *di.Config) (*di.Config, error) {
    for id, svc := range cfg.Services {
        // Only process method constructors
        if svc.Constructor == nil || svc.Constructor.Method == "" {
            continue
        }

        // Extract service name from ID (e.g., "log.database" → "database")
        parts := strings.Split(id, ".")
        if len(parts) < 2 {
            continue
        }
        channel := parts[len(parts)-1]

        // Add channel and channel name as variadic arguments
        svc.Constructor.Args = append(
            svc.Constructor.Args,
            di.Argument{
                Kind:  di.ArgLiteral,
                Literal: di.NewStringLiteral("channel"),
            },
            di.Argument{
                Kind:  di.ArgLiteral,
                Literal: di.NewStringLiteral(channel),
            },
        )

        cfg.Services[id] = svc
    }

    return cfg, nil
}
```

### Running the Example

```bash
cd examples/custom-pass

# Run custom generator
go run ./tools/gendi --config=./cmd/gendi.yaml --out=./cmd --pkg=main

# Run the application
go run ./cmd
```

### Example Output

```
Config before pass:
  Services: 2
  log.database: method constructor with 0 args
  log.auth: method constructor with 0 args

Config after pass:
  Services: 2
  log.database: method constructor with 2 args
  log.auth: method constructor with 2 args

Generated container with 2 services
```

## API Reference

### Core Types

```go
package di

// Config is the root configuration
type Config struct {
    Parameters map[string]Parameter
    Tags       map[string]Tag
    Services   map[string]Service
}

// Service represents a service definition
type Service struct {
    Type                string
    Constructor         *Constructor
    Alias               string
    Shared              bool
    Public              bool
    Tags                []ServiceTag
    Decorates           string
    DecorationPriority  int
}

// Constructor defines service constructor
type Constructor struct {
    Func   string
    Method string
    Args   []Argument
    SourceLoc *srcloc.Location
}

// Argument represents a constructor argument
type Argument struct {
    Kind      ArgumentKind
    Value     string
    Literal   Literal
    SourceLoc *srcloc.Location
}

// ArgumentKind enumerates argument types
type ArgumentKind int

const (
    ArgLiteral    ArgumentKind = iota
    ArgServiceRef
    ArgInner
    ArgParam
    ArgTagged
    ArgSpread
    ArgGoRef
    ArgFieldAccess
)

// Tag represents a tag definition
type Tag struct {
    ElementType   string
    SortBy        string
    Public        bool
    Autoconfigure bool
}

// ServiceTag represents a tag on a service
type ServiceTag struct {
    Name       string
    Attributes map[string]interface{}
}

// Parameter represents a parameter definition
type Parameter struct {
    Type  string
    Value interface{}
}
```

### Helper Functions

```go
// Create literals
func NewStringLiteral(s string) Literal
func NewIntLiteral(v int64) Literal
func NewFloatLiteral(v float64) Literal
func NewBoolLiteral(v bool) Literal
func NewNullLiteral() Literal
```

## See Also

- [Configuration Reference](./configuration.md)
- [Custom Pass Example](../examples/custom-pass/)
- [Technical Specification](./spec/README.md)
- [API Documentation](https://pkg.go.dev/github.com/asp24/gendi)
