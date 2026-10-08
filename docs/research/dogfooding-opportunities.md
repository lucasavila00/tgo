# TGo dogfooding opportunities

This note lists production changes that give useful TGo coverage. The compiler stays in Go under
the current [bootstrap policy](../bootstrap.md).

## Recommended work

### 1. Convert `internal/sourcefacts` to TGo

Affected files:

- [`internal/sourcefacts/index.go`](../../internal/sourcefacts/index.go)
- [`scripts/check_generated.py`](../../scripts/check_generated.py)

`Index` requires its file set and type information. `New` also requires a Go file. Convert the
package to `index.tgo` and use non-nil types for these values, the `Index` receiver, and syntax
arguments that all callers prove are present.

This is a good target because only the TGo linter imports the package. It tests non-nil facts across
`pkg/syntax`, `internal/sourcefacts`, and `internal/tgolint` without adding a compiler dependency.

The blocker is generated-output coverage. `check_generated.py` checks only `pkg/syntax` and
`internal/tgolint`. Add `internal/sourcefacts` and its build order in the same change. Keep optional
results, such as a missing function or expression, nullable.

### 2. Carry syntax non-nil facts into linter helpers

Affected files:

- [`pkg/syntax/ast.tgo`](../../pkg/syntax/ast.tgo)
- [`internal/tgolint/source_syntax.tgo`](../../internal/tgolint/source_syntax.tgo)

The syntax model now uses non-nil enum payloads and non-nil list elements. Private helpers such as
`sourceAssignment`, `sourceCall`, `sourceIdentifier`, `sourceGeneralDeclaration`, and
`sourceValueSpecification` still take nullable pointers.

Use non-nil parameters where every caller gets the value from a non-nil syntax list or a matched
payload. This is a good target because it tests exported facts between two TGo packages and removes
branches that cannot run.

The risk is an incorrect contract on a recovery or query path. Keep nullable inputs where nil is a
valid query value. Keep shape-query results nullable because `nil, false` is valid.

### 3. Replace `sourceModel` with an enum

Affected file:

- [`internal/tgolint/source_models.tgo`](../../internal/tgolint/source_models.tgo)

`sourceModel` stores `fields`, `base`, and `variants` in one struct. Only three shapes are valid:
a checked type has `base`, an enum has `variants`, and a struct has `fields`. `sourceDeclaration`
creates these shapes, and `sourceShapeMatches` finds the active shape from other fields.

A private enum with `Checked`, `Enum`, and `Struct` payloads removes the invalid combinations. This
is a good target because model verification processes real TGo source and needs exhaustive shape
handling.

Keep `modelWireFact` as a plain struct. It is the stable `go/analysis` wire format. The conversion
must not change fact data, source order, or diagnostic positions.

### 4. Trial postfix `!` in one parser path

Affected files:

- [`pkg/syntax/parser.tgo`](../../pkg/syntax/parser.tgo)
- [`pkg/syntax/parser_build.tgo`](../../pkg/syntax/parser_build.tgo)
- [`pkg/syntax/parser_extensions.tgo`](../../pkg/syntax/parser_extensions.tgo)

These files contain manual propagation after calls such as `closeToken`, `rawFields`,
`makeFields`, `parseField`, and `parseExpression`. One small call chain can test postfix `!` with
single and multiple non-error results.

The main risk is error text. Postfix `!` adds the static call name, while parser helpers often
preserve an exact syntax diagnostic or map its position. Use `!` only where the wrapper is the
intended public error. Keep manual branches that preserve or transform an error.

## Keep these areas in Go

- Keep [`internal/compiler`](../../internal/compiler),
  [`internal/outputname`](../../internal/outputname), and
  [`cmd/tgo/main.go`](../../cmd/tgo/main.go) in Go. They are in the compiler bootstrap path.
- Keep `modelWireFact` in [`internal/tgolint/models.tgo`](../../internal/tgolint/models.tgo) and the
  generic effect wire types in
  [`internal/tgolint/generic_effects.tgo`](../../internal/tgolint/generic_effects.tgo) as plain
  structs. Their decoders validate untrusted package facts before they create internal enums.
- Keep generated `*_tgo.go` files generated. Change their `.tgo` source or the emitter.
- Keep test fixtures under `testdata` out of production dogfooding decisions. They contain
  intentional output and error cases.
- Keep [`internal/compiler/sync_directory_unix.go`](../../internal/compiler/sync_directory_unix.go)
  and
  [`internal/compiler/sync_directory_other.go`](../../internal/compiler/sync_directory_other.go)
  in Go. They would test build constraints, but they are in the compiler. There is no current
  non-compiler production target for build-constraint dogfooding.
- Keep [`cmd/tgolint/main.go`](../../cmd/tgolint/main.go) in Go. It is a thin command wrapper and
  would not exercise a TGo model or safety feature.

Do the work in the listed order. Generated-output coverage must exist before a new production TGo
root is added.
