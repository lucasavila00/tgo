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
