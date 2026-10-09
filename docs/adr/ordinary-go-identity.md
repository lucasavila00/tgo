# Preserve ordinary Go bytes

Status: accepted

## Context

TGo accepts Go syntax and adds constructs that lower to Go. An accidental
rewrite of source that needs no lowering is a compiler defect. Source formatting
belongs to `tgofmt`.

Source usage policy runs in `tgolint`. It does not narrow the compiler's Go
compatibility.

## Decision

The compiler must accept valid Go in the selected build context. For each active
`.tgo` file that uses no TGo construct, it must write the input bytes directly
to its Go output. It must add no notice or ownership comment and must not call a
formatter.

This contract preserves spacing, comments, directives, build constraints,
explicit semicolons, line endings, and the presence or absence of a final
newline.

A contextual TGo construct counts as lowering even when Go can parse its tokens.
For example, `exhaustive:` has TGo behavior in a tag switch. A lowered file uses
the syntax-tree formatter and starts with Go's standard generated-code notice.
It has no custom TGo ownership comment.

Names of the forms `*_tgo.go` and `*_tgo_<target>.go` are reserved for compiler
output. This namespace lets the compiler replace and remove outputs without a
content marker or sidecar. It still refuses symlinks and other non-regular
outputs, preserves file modes, and restores changes after a failed build.

`tgolint` maps each reserved output name to the active source list from the
compiler. It compares the complete output with a new in-memory compilation
before it exports facts. Thus, a manual edit still fails verification.

## Tests

Normal CI checks three samples through the real `tgo build` path:

- folder fixtures cover non-gofmt layout, comments, directives, build
  constraints, explicit semicolons, CRLF, and a missing final newline;
- 64 valid programs come from a fixed seed and use small independent files; and
- a manifest selects 10 packages from the installed Go 1.27 source tree.

The Go corpus has 16 active source files. The test compares each output byte for
byte with its input and runs 9 same-package Go tests against both forms. It
excludes 9 external tests because they import the installed standard package.
The manifest is a small reproducible subset. It excludes packages that need cgo
or compiler-only source, or import Go `internal` packages that a copied module
cannot use.

These tests give bounded evidence. They do not prove the invariant for every Go
program. A failure prints the fixed seed, file name, source, and first changed
line. A fixed failure must become a folder fixture.

## Result

The prototype found that syntax-tree emission changed layout and removed a
source line directive. The exact-copy path removes that work. Lowered files keep
formatted code generation.
