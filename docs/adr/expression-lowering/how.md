# HOW

Issue: [#248](https://github.com/lucasavila00/tgo/issues/248)

- Use Go's existing parser and type checker. Do not write a parser or fork Go.
- Load source and type information with [`go/packages`][packages].
- Read the original AST (abstract syntax tree); build a new `go/ast` output.
- Adapt [Go's ordering pass][order] to that public AST.
- Format output with `go/format`; compile it with the Go toolchain.
- Do not restore the old compiler code.
- Pin Go and `golang.org/x/tools` versions when implementation starts.

[WHAT](what.md) defines the transformation. [PROOF](proof.md) checks it.

[packages]: https://pkg.go.dev/golang.org/x/tools/go/packages
[order]: https://go.dev/src/cmd/compile/internal/walk/order.go
