# Lower checked syntax into Go statements

Issue: [#248](https://github.com/lucasavila00/tgo/issues/248)

Use Go's standard parser through the frontend and `go/packages` for build
selection. The [source-checking decision](../source-checking/what.md) supplies
private checked types for TGo expressions.

Read the checked AST, or abstract syntax tree. Build output nodes without
changing the input tree. Use saved values to preserve evaluation order, branch
selection, assignment storage, and function return targets. These follow the
same ordering requirements as Go's compiler, implemented on public `go/ast`
nodes.

Format output with `go/format`. The driver writes final generated files and
checks them with stock Go. Pin Go and `golang.org/x/tools` versions.

See [the lowering explanation](../../arch/transpiler/correctness.md) for examples
and source files. [WHAT](what.md) states the behavior; [PROOF](proof.md) checks it.
