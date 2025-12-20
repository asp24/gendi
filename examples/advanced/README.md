# Advanced Example

This example demonstrates:
- configuration imports and overrides
- decorator chains with priority ordering
- tagged injection with interface element types
- method-based constructors

Files:
- `di.yaml` imports the base config and a region override.
- `services/base.yaml` defines the default services and tags.
- `services/region_us.yaml` overrides parameters and a service.

Generate the container:

```
go generate ./examples/advanced
```

The generated container lives in `examples/advanced/internal/di`.
