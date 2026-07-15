# Parameters

```yaml
parameters:
  dsn: "postgres://..."
  port: 8080
  timeout: 5s
```

Rules:
- A parameter declaration is a plain scalar default value (string, int, float, or bool); null values are rejected
- Parameters have no declared type: the target type is contextual, taken from each constructor argument the parameter is injected into
- Parameters are immutable
- Parameters are resolved at runtime via a `parameters.Provider` passed to the container

Generated containers include a constructor that accepts a provider. When no provider is passed, the container falls back to a map-backed provider built from the YAML values.

Runtime resolution is split into two steps:
- `Provider.Lookup(name)` returns the raw scalar value
- `parameters.Caster` converts it to the target type of the injection site (`ToInt`, `ToString`, `ToDuration`, ...)

Supported target types: `string`, `bool`, all signed and unsigned integer widths, `float32`, `float64`, `time.Duration`, `time.Time`, and named types with one of those underlying types (converted statically). `uintptr` and complex types are not supported.

Generation-time guarantees:
- A parameter injected into an unsupported target type fails generation
- Declared defaults are validated at generation time against every usage's target type by executing the standard caster
- Values supplied by a runtime provider are checked at construction time; errors carry the parameter name, service ID, argument index, and both the raw and target types

Notes:
- One runtime parameter may be requested as different types at different injection sites (e.g. `"42"` as both `string` and `int`)
- The default caster is `parameters.StandardCaster`; it can be replaced per container via the generated `With<Container>ParameterCaster` option
