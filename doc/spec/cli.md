# CLI

## Command

```
gendi
```

## Flags

| Flag | Description |
|----|------------|
| `--config` | Root YAML configuration file |
| `--out` | Output directory or file |
| `--pkg` | Go package name |
| `--container` | Container struct name |
| `--strict` | Enable strict validation (default: true) |
| `--build-tags` | Go build tags |
| `--enable-pass` | Enable an optional compiler pass by name; repeat for multiple passes; errors on unknown name |
| `--verbose` | Verbose logging |

## go:generate

```go
//go:generate go tool gendi --config=di.yaml --out=./internal/di --pkg=di
```
