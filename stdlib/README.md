# Standard Library Factories

Ready-to-use factory functions and service definitions for common Go standard library types.

## Installation

Import the stdlib services in your gendi configuration:

```yaml
imports:
  - github.com/asp24/gendi/stdlib/gendi.yaml
```

This imports pre-configured services for HTTP clients, loggers, and I/O.

## Quick Start

```yaml
imports:
  - github.com/asp24/gendi/stdlib/gendi.yaml

services:
  my_service:
    constructor:
      func: "github.com/myapp.NewService"
      args:
        - "@stdlib.http.client"  # Pre-configured HTTP client
        - "@stdlib.logger"       # Pre-configured logger
```

## Available Services

### HTTP Client

**Service ID:** `stdlib.http.client`

Pre-configured HTTP client with 30-second timeout.

```yaml
services:
  api_client:
    constructor:
      func: "github.com/myapp.NewAPIClient"
      args:
        - "@stdlib.http.client"
```

**Configuration:**
- Timeout: `%stdlib.http.timeout%` (default: `30s`)

### HTTP Client with Custom Transport

**Service ID:** `stdlib.http.client_with_transport`

HTTP client with customizable connection pooling.

```yaml
services:
  api_client:
    constructor:
      func: "github.com/myapp.NewAPIClient"
      args:
        - "@stdlib.http.client_with_transport"
```

**Configuration:**
- Timeout: `%stdlib.http.timeout%` (default: `30s`)
- Max idle connections: `%stdlib.http.max_idle_conns%` (default: `100`)
- Max idle per host: `%stdlib.http.max_idle_conns_per_host%` (default: `10`)
- Idle timeout: `%stdlib.http.idle_conn_timeout%` (default: `90s`)

### HTTP Transport

**Service ID:** `stdlib.http.transport`

Standalone HTTP transport for custom clients.

```yaml
services:
  custom_client:
    constructor:
      func: "github.com/myapp.NewCustomHTTPClient"
      args:
        - "@stdlib.http.transport"
```

### Logger (slog)

**Service ID:** `stdlib.logger` (alias: `stdlib.slog`)

Structured logger using `log/slog` with text handler to stdout.

```yaml
services:
  app:
    constructor:
      func: "github.com/myapp.NewApp"
      args:
        - "@stdlib.logger"
```

**Configuration:**
- Log level: `%stdlib.slog.level%` (default: `Info`)

**Available log levels:**
- `Debug` (-4)
- `Info` (0)
- `Warn` (4)
- `Error` (8)

### Log Handlers

**Service IDs:**
- `stdlib.slog.handler.text` - Text format handler
- `stdlib.slog.handler.json` - JSON format handler

Custom logger with JSON output:

```yaml
services:
  json_logger:
    constructor:
      func: "log/slog.New"
      args:
        - "@stdlib.slog.handler.json"
```

### I/O Writers

**Service IDs:**
- `stdlib.stdout` - Standard output (`os.Stdout`)
- `stdlib.stderr` - Standard error (`os.Stderr`)

```yaml
services:
  file_logger:
    constructor:
      func: "myapp.NewFileLogger"
      args:
        - "@stdlib.stdout"
```

## Factory Functions

The stdlib package provides factory functions you can use directly in your service definitions.

### Channels

**`NewChan[T](size int) chan T`**

Creates a buffered channel of any type.

```yaml
services:
  events:
    constructor:
      func: "github.com/asp24/gendi/stdlib.NewChan[github.com/myapp.Event]"
      args:
        - 100  # Buffer size
    public: true
```

Generated code:
```go
func (c *Container) buildEvents() (chan myapp.Event, error) {
    return stdlib.NewChan[myapp.Event](100), nil
}
```

**Use cases:**
- Event channels
- Work queues
- Message passing

### HTTP

**`NewHTTPClient(timeout time.Duration) *http.Client`**

Creates HTTP client with timeout.

```yaml
services:
  fast_client:
    constructor:
      func: "github.com/asp24/gendi/stdlib.NewHTTPClient"
      args:
        - 5000000000  # 5 seconds in nanoseconds
```

**`NewHTTPClientWithTransport(timeout time.Duration, transport *http.Transport) *http.Client`**

Creates HTTP client with custom transport.

```yaml
services:
  custom_client:
    constructor:
      func: "github.com/asp24/gendi/stdlib.NewHTTPClientWithTransport"
      args:
        - "%http_timeout%"
        - "@custom_transport"
```

**`NewHTTPTransport(maxIdleConns, maxIdleConnsPerHost int, idleConnTimeout time.Duration) *http.Transport`**

Creates HTTP transport with connection pooling.

```yaml
services:
  high_perf_transport:
    constructor:
      func: "github.com/asp24/gendi/stdlib.NewHTTPTransport"
      args:
        - 200   # Max idle connections
        - 20    # Max idle per host
        - 120000000000  # 120 seconds
```

### Logging (slog)

**`NewSlogTextHandler(w io.Writer, level slog.Level) *slog.TextHandler`**

Creates text format log handler.

```yaml
services:
  text_handler:
    constructor:
      func: "github.com/asp24/gendi/stdlib.NewSlogTextHandler"
      args:
        - "@stdlib.stdout"
        - 0  # Info level
```

**`NewSlogJSONHandler(w io.Writer, level slog.Level) *slog.JSONHandler`**

Creates JSON format log handler.

```yaml
services:
  json_handler:
    constructor:
      func: "github.com/asp24/gendi/stdlib.NewSlogJSONHandler"
      args:
        - "@stdlib.stderr"
        - -4  # Debug level
```

**`NewSlogLogger(handler slog.Handler) *slog.Logger`**

Creates logger from handler.

```yaml
services:
  custom_logger:
    constructor:
      func: "github.com/asp24/gendi/stdlib.NewSlogLogger"
      args:
        - "@custom_handler"
```

### I/O

**`NewStdout() io.Writer`**

Returns `os.Stdout`.

```yaml
services:
  stdout:
    constructor:
      func: "github.com/asp24/gendi/stdlib.NewStdout"
```

**`NewStderr() io.Writer`**

Returns `os.Stderr`.

```yaml
services:
  stderr:
    constructor:
      func: "github.com/asp24/gendi/stdlib.NewStderr"
```

### Slices

**`NewSlice[T]() []T`**

Creates an empty slice of any type.

```yaml
services:
  handlers:
    constructor:
      func: "github.com/asp24/gendi/stdlib.NewSlice[github.com/myapp.Handler]"
```

## Parameter Overrides

Override default parameters in your configuration:

```yaml
imports:
  - github.com/asp24/gendi/stdlib/gendi.yaml

parameters:
  # Override HTTP timeout
  stdlib.http.timeout:
    type: time.Duration
    value: "60s"

  # Override log level
  stdlib.slog.level:
    type: int
    value: -4  # Debug

  # Override connection pool settings
  stdlib.http.max_idle_conns:
    type: int
    value: 200

  stdlib.http.max_idle_conns_per_host:
    type: int
    value: 20

  stdlib.http.idle_conn_timeout:
    type: time.Duration
    value: "120s"
```

## Default Parameters

The stdlib module defines these parameters:

```yaml
parameters:
  stdlib.http.timeout:
    type: time.Duration
    value: "30s"

  stdlib.http.max_idle_conns:
    type: int
    value: 100

  stdlib.http.max_idle_conns_per_host:
    type: int
    value: 10

  stdlib.http.idle_conn_timeout:
    type: time.Duration
    value: "90s"

  stdlib.slog.level:
    type: int
    value: 0  # Info
```

## Complete Service Definitions

The stdlib `gendi.yaml` file contains:

```yaml
parameters:
  stdlib.http.timeout:
    type: time.Duration
    value: "30s"
  stdlib.http.max_idle_conns:
    type: int
    value: 100
  stdlib.http.max_idle_conns_per_host:
    type: int
    value: 10
  stdlib.http.idle_conn_timeout:
    type: time.Duration
    value: "90s"
  stdlib.slog.level:
    type: int
    value: 0

services:
  stdlib.http.transport:
    constructor:
      func: "$this.NewHTTPTransport"
      args:
        - "%stdlib.http.max_idle_conns%"
        - "%stdlib.http.max_idle_conns_per_host%"
        - "%stdlib.http.idle_conn_timeout%"
    shared: true

  stdlib.http.client_with_transport:
    constructor:
      func: "$this.NewHTTPClientWithTransport"
      args:
        - "%stdlib.http.timeout%"
        - "@stdlib.http.transport"
    shared: true

  stdlib.http.client:
    constructor:
      func: "$this.NewHTTPClient"
      args:
        - "%stdlib.http.timeout%"
    shared: true

  stdlib.stdout:
    constructor:
      func: "$this.NewStdout"
    shared: true

  stdlib.stderr:
    constructor:
      func: "$this.NewStderr"
    shared: true

  stdlib.slog.handler.text:
    constructor:
      func: "$this.NewSlogTextHandler"
      args:
        - "@stdlib.stdout"
        - "%stdlib.slog.level%"
    shared: true

  stdlib.slog.handler.json:
    constructor:
      func: "$this.NewSlogJSONHandler"
      args:
        - "@stdlib.stdout"
        - "%stdlib.slog.level%"
    shared: true

  stdlib.slog:
    constructor:
      func: "$this.NewSlogLogger"
      args:
        - "@stdlib.slog.handler.text"
    shared: true

  stdlib.logger:
    alias: "stdlib.slog"
```

## Examples

### HTTP Client with Custom Timeout

```yaml
imports:
  - github.com/asp24/gendi/stdlib/gendi.yaml

parameters:
  stdlib.http.timeout:
    type: time.Duration
    value: "5s"

services:
  api_client:
    constructor:
      func: "github.com/myapp.NewAPIClient"
      args:
        - "@stdlib.http.client"
```

### JSON Logger with Debug Level

```yaml
imports:
  - github.com/asp24/gendi/stdlib/gendi.yaml

parameters:
  stdlib.slog.level:
    type: int
    value: -4  # Debug

services:
  logger:
    constructor:
      func: "log/slog.New"
      args:
        - "@stdlib.slog.handler.json"
    public: true
```

### Event Channel

```yaml
services:
  order_events:
    constructor:
      func: "github.com/asp24/gendi/stdlib.NewChan[github.com/myapp/orders.OrderEvent]"
      args:
        - 100  # Buffer size
    public: true

  order_processor:
    constructor:
      func: "github.com/myapp/orders.NewProcessor"
      args:
        - "@order_events"
```

### High-Performance HTTP Client

```yaml
imports:
  - github.com/asp24/gendi/stdlib/gendi.yaml

parameters:
  stdlib.http.timeout:
    type: time.Duration
    value: "60s"

  stdlib.http.max_idle_conns:
    type: int
    value: 500

  stdlib.http.max_idle_conns_per_host:
    type: int
    value: 50

services:
  api_client:
    constructor:
      func: "github.com/myapp.NewAPIClient"
      args:
        - "@stdlib.http.client_with_transport"
```

## Testing with Stdlib

The stdlib services are shared singletons. For testing, override them in your test configuration:

```yaml
# test/gendi.yaml
imports:
  - ../gendi.yaml

services:
  # Override HTTP client with mock
  stdlib.http.client:
    constructor:
      func: "github.com/myapp/mocks.NewHTTPClient"
    shared: true

  # Override logger with no-op
  stdlib.logger:
    constructor:
      func: "github.com/myapp/mocks.NewNoopLogger"
    shared: true
```

## See Also

- [Configuration Reference](../doc/configuration.md)
- [API Documentation](https://pkg.go.dev/github.com/asp24/gendi/stdlib)
- [Examples](../examples/)
