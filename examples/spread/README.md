# Spread Operator Example

This example demonstrates the spread operator (`!spread:`) feature in gendi, which unpacks slice values into variadic parameters.

## Features Demonstrated

1. **Spread Tagged Injection** - Unpack tagged services into variadic parameters
2. **Spread Service Reference** - Unpack a service that returns a slice
3. **Mixed Arguments** - Combine regular arguments with spread

## Spread Operator Syntax

The spread operator uses `!spread:` prefix:
- `!spread:@service` - Spread a service that returns a slice
- `!spread:!tagged:tag` - Spread tagged injection

### Example 1: Spread Tagged Injection

```yaml
services:
  handler.home:
    tags:
      - name: handler
        priority: 100

  handler.api:
    tags:
      - name: handler
        priority: 200

  server:
    constructor:
      func: "app.NewServer"  # NewServer(handlers ...Handler)
      args:
        - "!spread:!tagged:handler"  # Unpacks []Handler into ...Handler
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

### Example 2: Spread Service Reference

```yaml
services:
  all_handlers:
    constructor:
      func: "app.GetAllHandlers"  # Returns []Handler

  server:
    constructor:
      func: "app.NewServer"  # NewServer(handlers ...Handler)
      args:
        - "!spread:@all_handlers"  # Unpacks []Handler into ...Handler
```

Generated code:
```go
func (c *Container) buildServer() (*Server, error) {
    allHandlers, err := c.getAllHandlers()
    if err != nil {
        return nil, err
    }
    return app.NewServer(allHandlers...), nil
}
```

### Example 3: Mixed Arguments

```yaml
services:
  server:
    constructor:
      func: "app.NewPrefixedServer"  # NewPrefixedServer(prefix string, handlers ...Handler)
      args:
        - "%server_prefix%"            # Regular parameter
        - "!spread:!tagged:handler"    # Spread as last argument
```

Generated code:
```go
func (c *Container) buildServer() (*PrefixedServer, error) {
    prefix, err := c.params.GetString("server_prefix")
    if err != nil {
        return nil, err
    }
    tagged_1, err := c.getTaggedWithHandler()
    if err != nil {
        return nil, err
    }
    return app.NewPrefixedServer(prefix, tagged_1...), nil
}
```

## Rules

- Only one spread allowed per constructor
- Spread must be the last argument
- Inner value must be a slice type
- Target parameter must be variadic

## Running the Example

Generate the container:

```bash
cd examples/spread
go generate
```

Run the example:

```bash
go run .
```

Expected output:
```
=== Spread Operator Example ===

This example demonstrates three use cases of the spread operator:
1. Spreading tagged injection into variadic parameters
2. Spreading service references into variadic parameters
3. Mixing regular arguments with spread

--- Example 1: Spread Tagged Injection ---
Config: !spread:!tagged:handler

Registered 3 handlers:
  1. Priority: 200
  2. Priority: 100
  3. Priority: 50

Server handling request: /users
[APIHandler priority=200] Handling: /users
[HomeHandler priority=100] Handling: /users
[AdminHandler priority=50] Handling: /users

--- Example 2: Spread Service Reference ---
Config: !spread:@all_handlers

Registered 3 handlers:
  1. Priority: 100
  2. Priority: 200
  3. Priority: 50

Server handling request: /products
[HomeHandler priority=100] Handling: /products
[APIHandler priority=200] Handling: /products
[AdminHandler priority=50] Handling: /products

--- Example 3: Mixed Arguments with Spread ---
Config: prefix parameter + !spread:!tagged:handler

PrefixedServer (prefix=/api/v1) has 3 handlers:
  1. Priority: 200
  2. Priority: 100
  3. Priority: 50

PrefixedServer handling request: /api/v1/orders
[APIHandler priority=200] Handling: /api/v1/orders
[HomeHandler priority=100] Handling: /api/v1/orders
[AdminHandler priority=50] Handling: /api/v1/orders

=== Example Complete ===
```

## Use Cases

The spread operator is useful when:
- Working with variadic constructors that accept multiple services
- Building servers/routers that register multiple handlers
- Creating aggregators that collect and pass through tagged services
- Avoiding manual unpacking of slice-based dependencies
