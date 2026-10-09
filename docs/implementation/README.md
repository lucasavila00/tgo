# Implementation notes

This document records repository rules for work on the TGo compiler and analyzer. These rules are
not language behavior.

## Bootstrap

The tgo compiler is implemented in Go. A compiler build does not require an existing tgo
compiler.

Repository tools outside the compiler bootstrap path can use tgo. For example, `tgolint` uses
tgo source files.

Each committed `.tgo` source file has a generated `*_tgo.go` file in the same directory. The Go
tool uses these generated files as normal Go source. `make generated` verifies that the committed
files match the current compiler output.

## Compiler boundary

`internal/compiler` is the Go bootstrap core. Its job ends when it returns ordinary Go source and
diagnostics that are required to produce that source. It may:

- parse TGo and build its Go projection;
- type-check only when lowering or generated representation needs type facts;
- lower TGo syntax while preserving Go evaluation order;
- validate generated names, wire layouts, and representations; and
- emit formatted Go source.

Source policy, modernization advice, navigation, generated-output integrity, package selection,
locking, file transactions, and stale-output cleanup stay outside the compiler. Production code
for these tasks must be `.tgo` and must use `pkg/syntax` instead of `go/ast`.

The remaining exceptions have migration issues:

| Compiler code | Destination | Issue |
| --- | --- | --- |
| `check.go` model usage policy | `tgolint` | [#61](https://github.com/lucasavila00/go2/issues/61) |
| `enum_switch.go` switch policy | `tgolint` | [#60](https://github.com/lucasavila00/go2/issues/60) |
| `analysis.go` tooling | TGo analysis package | [#63](https://github.com/lucasavila00/go2/issues/63) |
| `build.go` and file transaction helpers | TGo build driver | [#64](https://github.com/lucasavila00/go2/issues/64) |
| `verify.go` verification | Remove from compiler | [#65](https://github.com/lucasavila00/go2/issues/65) |

## Syntax boundary

The public `pkg/syntax` tree is the common source model for tools that analyze TGo code. Downstream
packages must use its nodes and must not import or expose `go/ast` types. Only the compiler and the
private parser and converter in `pkg/syntax` can use `go/ast`.

Run `make ast-boundary` to check this boundary.

## Enum storage

The compiler starts with each payload in inline storage. It uses the 64-bit Go layout to calculate
the enum size. While the size exceeds 80 bytes, it boxes the largest inline payload. An equal size
selects the first declared variant. Boxed variants share one interface field. Empty variants add no
payload storage.

## Repository checks

Production source in this repository does not use `tgolint` suppression directives. The main CI
target checks generated files, the syntax boundary, allocation budgets, Markdown, lint, and tests.
