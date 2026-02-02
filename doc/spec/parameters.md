# Parameters

```yaml
parameters:
  dsn:
    type: string
    value: "postgres://..."
```

Rules:
- `type` and `value` are mandatory
- Type is used for Go code generation
- Parameters are immutable
- Parameters are resolved at runtime via a `parameters.Provider` passed to the container

Generated containers include a constructor that accepts a provider. When no provider is passed, the container falls back to a map-backed provider built from the YAML values.

Supported parameter types:
- `string`
- `int`
- `bool`
- `float64`
- `time.Duration`

Notes:
- `time.Duration` values are defined as YAML strings (for example, `"1s"`)
- Generated code calls `Provider.GetDuration` for `time.Duration` parameters
