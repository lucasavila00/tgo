# HOW

Issue: [#248](https://github.com/lucasavila00/tgo/issues/248)

## Terms

- **AST:** abstract syntax tree; Go's structured representation of source code.
- **Lowering:** rewriting source constructs into simpler constructs.
- **Temporary variable:** a generated local that saves an evaluated value.

## Toolchain decision

- Use Go's public parser, AST, type checker, and formatter.
- Load packages with [`go/packages`][packages], requesting source ASTs and type
  information. It handles imports and the module's build environment.
- Use the standard [`go/parser`][parser] through that loader. Do not write a
  parser for ordinary Go or fork the Go compiler.
- Keep the original AST and `go/types` information unchanged. Build a separate
  output AST, then format it with `go/format`.
- Compile and execute generated files with the Go toolchain. That independently
  checks whether the output is valid Go.
- Study [Go's ordering pass][order] for evaluation-order rules. Adapt necessary
  algorithms to the public AST; do not import compiler-internal packages.

## Location and dependencies

- Create a separate module in `spikes/expressionlowering/`.
- Keep fixture programs and the proof runner in that module.
- Pin Go and `golang.org/x/tools` versions when implementation starts.
- Do not restore the removed compiler or connect this spike to production.

## Alternatives

- **Custom parser:** unnecessary for #248, which takes ordinary Go input.
- **Go compiler fork:** adds a compiler tree and maintenance cost before the
  insertion algorithm is proven.
- **TGo parser work:** belongs to #247. Its syntax decision must not change
  this spike's Go input contract.

Behavior and algorithm: [WHAT](what.md). Validation plan: [PROOF](proof.md).

[packages]: https://pkg.go.dev/golang.org/x/tools/go/packages
[parser]: https://pkg.go.dev/go/parser
[order]: https://go.dev/src/cmd/compile/internal/walk/order.go
