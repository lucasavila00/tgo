# TGo

TGo adds compile-time checks and shorter error handling to Go. The
[language vision](docs/vision.md) describes the intended language and syntax.

## Current transpiler

The transpiler emits ordinary Go source from `.tgo` files. It supports:

- `call()!!`: return the original error from the current function.
- `call()!`: add the call name to the error and preserve its identity through `%w`.
- Statement insertion after an expression through the Go API.

Lowering preserves evaluation order, scope, conditional execution, and deferred
calls. Module compilation uses Go build selection, including test packages.

See [transpiler use and API](docs/transpiler.md) for commands and examples.

## Repository layout

- `cmd/tgo`: command-line tool.
- `pkg/transpiler`: public Go API.
- `internal`: compiler implementation.
- `tests`: integration tests, fixtures, and generated Go snapshots.
- `docs`: language vision, architecture, and design decisions.

The repository uses one Go module at its root. See the
[architecture](docs/arch/transpiler.md) for package dependencies and the
[documentation index](docs/README.md) for design plans and project goals.
