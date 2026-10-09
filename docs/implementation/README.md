# Implementation notes

This document records repository rules for work on the TGo compiler and analyzer. These rules are
not language behavior.

## Bootstrap

The tgo compiler is implemented in Go. A compiler build does not require an existing tgo
compiler.

Repository tools outside the compiler bootstrap path can use tgo. For example, `tgolint` uses
tgo source files.

Each production `.tgo` source file has a generated `*_tgo.go` file in the same directory. The Go
tool uses these generated files as normal Go source. `make generated` uses the build driver's
package and output rules to verify that the committed files match the current compiler output.
It skips hidden directories, underscore-prefixed directories, `vendor`, `testdata`, nested modules,
temporary `bin` output, and the copied Go corpus in `third_party/go`.

## Ordinary Go source identity

The driver copies an active `.tgo` file byte for byte when it has no TGo
construct. It does not format the file or add a generated-code notice. A
contextual TGo construct, such as `exhaustive:` in a tag switch, selects the
lowering path even when the Go parser can accept its tokens.

Focused fixtures protect comments, directives, build constraints, semicolons,
line endings, and final-newline behavior. A fixed-seed source sample adds
ordinary Go shapes. A manifest selects a bounded Go standard-library corpus for
byte comparison and same-package tests. Add each fixed identity failure as a
focused fixture.

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

The remaining exception has a migration issue. The pull request that completes
the migration must remove its row. Remove this table when no exception remains.

| Compiler code | Destination | Issue |
| --- | --- | --- |
| [`check.go`](../../internal/compiler/check.go) | `tgolint` | [#61](https://github.com/lucasavila00/tgo/issues/61) |

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

## Code navigation

The VS Code extension uses a TextMate grammar for token colors and direct VS
Code providers for read-only navigation. The providers use the JSON-lines
protocol of the bundled `tgonav` helper. The helper is written in TGo and uses
`pkg/syntax` and shared source analysis. It does not copy the parser. Add an LSP
adapter only when another editor needs the same operations.

The helper accepts file URIs and UTF-8 byte offsets and returns exact source
ranges. The extension converts between byte offsets and VS Code UTF-16
positions. The navigation index uses type-object identity instead of name
matching. It returns no result when it has no exact range. It maps generated
members back to their owning TGo declaration.

The index follows the active Go build configuration. A change to `.tgo`, `.go`,
`go.mod`, or `go.work` invalidates the workspace index. Helper tests exercise
the JSON protocol and semantic results. VS Code integration tests cover
provider registration, helper startup, cancellation, and range conversion.

## Repository checks

Production source in this repository does not use `tgolint` suppression directives. The main CI
target checks generated files, the syntax boundary, allocation budgets, Markdown, lint, and tests.

Run `make install-hooks` to install the repository pre-commit hook. The hook checks staged
whitespace and Markdown line width only. It does not build the repository or run tests. Hosted CI
runs the complete validation suite after a branch is pushed.

The [Go printer port](go-printer-port.md) defines the staged formatter replacement.
