# Custom Compiler Pass Example

This example demonstrates how to create and use custom compiler passes with gendi using a custom generator binary.

## Project Structure

```
examples/custom-pass/
├── tools/gendi/          # Custom generator with passes
│   └── main.go
├── cmd/                  # Application entry point & config
│   ├── main.go          # Demo application
│   ├── slog.go          # Structured logger setup
│   ├── gendi.yaml       # Root DI configuration
│   └── container_gen.go # Generated container
├── internal/
│   ├── app/             # Application code
│   │   ├── http.go      # Server and HTTPHandler interface
│   │   ├── user.go      # User repository and handler
│   │   ├── product.go   # Product repository and handler
│   │   └── gendi.yaml   # Service definitions
│   └── di/              # Custom compiler passes
│       ├── autotag_pass.go
│       └── slog_pass.go
```

## Custom Passes

### 1. AutoTagPass

Automatically tags services based on naming conventions:
- Services ending with `.handler` → `http.handler` tag
- Services ending with `.repo` → `repository` tag

**Benefits:** Convention over configuration - services are automatically collected by tags without manual tagging.

### 2. SLogPass

Creates channel-specific structured loggers for services tagged with `slog`:
- Looks for `slog` tag with `channel` attribute
- Creates a non-shared named logger using `@logger.With("channel", channelName)`
- Replaces `@logger` dependency with the channel-specific logger

**Example:**
```yaml
services:
  user.handler:
    constructor:
      func: "$this.NewUserHandler"
      args:
        - "@logger"      # Will be replaced with @user.handler.logger
        - "@user.repo"
    tags:
      - { name: "slog", channel: "user" }
```

The pass transforms this to inject `@user.handler.logger` (which is `@logger.With("channel", "user")`).

**How it works:**
1. Finds services tagged with `slog` and a `channel` attribute
2. Creates a new service `<service-id>.logger` that calls `@logger.With("channel", "<channel-name>")`
3. Replaces all `@logger` references in the service's constructor args with the channel-specific logger
4. Sets the generated logger as non-shared (each call creates a new instance)

This pass demonstrates **variadic function support** - `slog.Logger.With(args ...any)` accepts variable arguments.

## How It Works

### 1. Define Custom Passes

Implement the `di.Pass` interface:

```go
type SLogPass struct{}

func (s *SLogPass) Name() string {
    return "slog"
}

func (s *SLogPass) Process(cfg *di.Config) (*di.Config, error) {
    // Transform config and return modified version
    return cfg, nil
}
```

### 2. Create Custom Generator Binary

`tools/gendi/main.go`:
```go
func main() {
    passes := []gendi.Pass{
        &di.AutoTagPass{},
        &di.SLogPass{},
    }

    if err := cmd.Run(flag.CommandLine, passes); err != nil {
        fmt.Fprintf(os.Stderr, "%v\n", err)
        os.Exit(1)
    }
}
```

### 3. Run Generation

You can generate the container in two ways:

**Option 1: Direct command**
```bash
go run ./tools/gendi --config=./cmd/gendi.yaml --out=./cmd --pkg=main
```

**Option 2: Using go:generate**
```bash
cd cmd
go generate
```

The `cmd/main.go` includes a go:generate directive:
```go
//go:generate go run ../tools/gendi/main.go --config=gendi.yaml --out=. --pkg=main
```

## Benefits

- **Convention over configuration**: Auto-tag services based on naming patterns
- **Structured logging**: Automatic channel-specific logger injection
- **Modularity**: Services defined in `internal/app/gendi.yaml` via glob imports
- **Type safety**: All transformations happen at generation time with full type checking
- **Variadic support**: Passes can use methods with variable arguments like `logger.With(args...)`

## Run the Example

```bash
# Generate the container with custom passes
go run ./tools/gendi --config=./cmd/gendi.yaml --out=./cmd --pkg=main

# Run the demo
go run ./cmd/*.go
```

**Example Output:**
```
level=INFO msg="Starting server" channel=http handlers=2
level=INFO msg="ProductHandler: handling request" channel=product
level=INFO msg="Product #123 from postgres://localhost/myapp" channel=http handler=0
level=INFO msg="UserHandler: handling request" channel=user
level=INFO msg="User #42 from postgres://localhost/myapp" channel=http handler=1
```

Notice how each service gets its own channel-specific logger automatically:
- Server uses `channel=http`
- Product handler uses `channel=product`
- User handler uses `channel=user`

## Key Features Demonstrated

1. **Custom generator binary** - `tools/gendi/` with project-specific passes
2. **Auto-tagging** - Services automatically tagged by naming convention
3. **Structured logging** - Channel-specific loggers injected automatically via SLogPass
4. **Glob imports** - Services loaded from `internal/**/gendi.yaml`
5. **Tagged injection** - `!tagged:http.handler` collects all auto-tagged handlers
6. **Variadic methods** - SLogPass uses `@logger.With("channel", "name")` with multiple arguments
7. **go:generate** - Code generation integrated with standard Go tooling
8. **Non-shared services** - Generated channel loggers are created fresh on each access
