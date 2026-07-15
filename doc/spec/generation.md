# Generation

## Container Structure

```go
type Container struct {
    // private fields
}
```

Constructor:
```go
func NewContainer(params parameters.Provider, opts ...ContainerOption) *Container
```

Options:
```go
func WithContainerErrorHandler(handler func(serviceName string, err error)) ContainerOption
func WithContainerParameterCaster(caster parameters.Caster) ContainerOption
```

The container stores a `parameters.Caster` (default `parameters.StandardCaster`)
and converts each looked-up parameter to the target type of its injection site.

## Getter Methods

- `GetX() (T, error)`
- `MustX() T` — panics on error and optionally reports via container error handler

All getters are strictly typed.

## Shared vs Non-shared

- Shared: lazy singleton
- Non-shared: new instance per call

## Error Reporting

The generator emits diagnostic errors containing:
- service ID
- configuration field
- expected vs actual type
- dependency chain when applicable

Example:
```
service "payments":
  constructor "NewService":
  arg[0]: expected []payments.Provider, got []any
```

## Circular Dependency Detection

- Cycles are detected at generation time
- Generation fails
- Errors include the dependency trace
