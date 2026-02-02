# Configuration

## Root Structure

```yaml
imports: []
parameters: {}
tags: {}
services: {}
```

## Imports

Imports allow configuration decomposition, reuse, and overriding.

Format:
```yaml
imports:
  - ./services/db.yaml
  - ./services/*.yaml
  - path: github.com/acme/billing/config/*.yaml
```

Rules:
- Imports are processed in declaration order
- Relative paths are resolved relative to the importing file
- Recursive imports are allowed; cyclic imports are forbidden
- Later definitions override earlier ones
- Imports can be a string path or a mapping with `path`
- Module imports resolve to `gendi.yaml`/`gendi.yml` at module root when no file is provided
- Glob patterns are supported; matches are expanded in lexicographic order

Import exclusions:
```yaml
imports:
  - path: ./services/*.yaml
    exclude:
      - ./services/test_*.yaml
      - ./services/internal/*.yaml
```

## The `$this` Package Token

`$this` can be used in service `type`, constructor `func`, and `method` fields to reference the Go package where the configuration file is located.

Usage:
```yaml
services:
  logger:
    type: "*$this.Logger"
    constructor:
      func: "$this.NewLogger"
```

Resolution:
- `$this` is resolved by locating the nearest `go.mod` and computing the package path relative to module root
- Each imported config file has its own `$this` context based on its location

Rules:
- For `type` field: `$this.` can appear anywhere in the type (`*$this.T`, `[]$this.T`, `map[K]$this.V`)
- For `func` and `method` fields: `$this` must appear at the start of the path
- If package resolution fails, `$this` remains unchanged and will cause a generation error if the symbol is not found

## Constructor Arguments

| Syntax | Meaning |
| --- | --- |
| `@service.id` | Reference to another service |
| `@.inner` | Inner service in a decorator |
| `%param.name%` | Parameter reference |
| `!tagged:tag.name` | Tagged injection |
| `!spread:@service` | Spread slice into variadic parameters |
| `!spread:!tagged:tag` | Spread tagged collection into variadic parameters |
| literal | YAML scalar literal |

Argument count and types are strictly validated.

### Spread Operator

The `!spread:` operator unpacks slice values into variadic parameters using Go's `...` syntax.

Example:
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
- Works with both service references and tagged injection

## Literals

Constructor arguments can be YAML scalar literals:
- string
- int
- float
- bool
- null
