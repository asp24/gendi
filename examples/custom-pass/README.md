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
│   │   ├── app.go       # Repositories, handlers, server
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
- Creates a named logger using `@logger.With(channel)`
- Replaces `@logger` dependency with the channel-specific logger

**Example:**
```yaml
services:
  user.handler:
    constructor:
      func: "$this.NewUserHandler"
      args:
        - "@logger"
    tags:
      - { name: "slog", channel: "users" }
```

The pass transforms this to inject `@user.handler.logger` (which is `@logger.With("users")`).

## How It Works

### 1. Define Custom Passes

Implement the `di.Pass` interface:

```go
type SLogPass struct{}

func (s *SLogPass) Name() string {
    return "slog"
}

func (s *SLogPass) Process(cfg *di.Config) (*di.Config, error) {
    // Transform config
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

    if err := cli.Run(flag.CommandLine, passes); err != nil {
        fmt.Fprintf(os.Stderr, "%v\n", err)
        os.Exit(1)
    }
}
```

### 3. Run Generation

```bash
go run ./tools/gendi --config=./cmd/gendi.yaml --out=./cmd --pkg=main
```

## Benefits

- **Convention over configuration**: Auto-tag services based on naming patterns
- **Structured logging**: Automatic channel-specific logger injection
- **Modularity**: Services defined in `internal/app/gendi.yaml` via glob imports
- **Type safety**: All transformations happen at generation time with full type checking

## Run the Example

```bash
# Generate the container with custom passes
go run ./tools/gendi --config=./cmd/gendi.yaml --out=./cmd --pkg=main

# Run the demo
go run ./cmd/*.go
```

**Example Output:**
```
level=INFO msg="UserHandler: handling request" channel=users
level=INFO msg="ProductHandler: handling request" channel=product
Server starting with 2 handlers
  Handler 0: Product #123 from postgres://localhost/myapp
  Handler 1: User #42 from postgres://localhost/myapp
```

Notice how each handler gets its own channel-specific logger automatically!

## Key Features Demonstrated

1. **Custom generator binary** - `tools/gendi/` with project-specific passes
2. **Auto-tagging** - Services automatically tagged by naming convention
3. **Structured logging** - Channel-specific loggers injected automatically
4. **Glob imports** - Services loaded from `internal/**/gendi.yaml`
5. **Tagged injection** - `!tagged:http.handler` collects all auto-tagged handlers
