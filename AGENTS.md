# Repository Guidelines

## Project Structure & Module Organization
- `cmd/gendi/` hosts the CLI generator entry point.
- `internal/generator/` contains the core loader, validation, and codegen logic.
- `examples/basic/` and `examples/advanced/` show runnable configs, apps, and generated output under `internal/di/`.
- Top-level files like `config.go` and `config_os.go` define shared configuration structures.

## Build, Test, and Development Commands
- `go run ./cmd/gendi --config=examples/basic/gendi.yaml --out=examples/basic/internal/di --pkg=di`: generate a container from YAML.
- `go generate ./...`: run any `//go:generate` hooks (see README for tool setup).
- `go test ./...`: run unit and integration tests.
- `go build ./cmd/gendi`: build the generator binary.
- `just gen-examples`: regenerate example containers (requires `just`).

## Coding Style & Naming Conventions
- Use standard Go formatting (`gofmt`) and idiomatic Go naming (CamelCase for exported symbols, lower-case for package names).
- Keep files and tests in the same package when possible; use `*_test.go` naming for tests.
- Generated files follow `*_gen.go` naming (e.g., `container_gen.go`) and should not be edited by hand.

## Testing Guidelines
- Tests live in `internal/generator/` with golden-style expectations (see `generator_test.go`).
- Prefer table-driven tests and check both error messages and generated output when relevant.
- Run `go test ./...` locally before submitting changes.

## Commit & Pull Request Guidelines
- Commit messages in this repo are short, imperative, and unscoped (e.g., “Tests”, “Rename project to gendi”).
- PRs should include a concise summary, the reasoning behind changes, and any updates to examples or docs when behavior changes.

## Configuration & Generation Notes
- YAML configs are expected in files like `gendi.yaml`; imports are relative to the importing file.
- Keep example configs and generated output in sync when updating generator behavior.
