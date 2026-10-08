# Superset notes

- [ ] Fix a target Go version for the first compiler and its compatibility corpus.
- [ ] Define parser rules for the error branch, including semicolon insertion.
- [ ] Check scope with outer `err`, named returns, partial results, and shadowing.
- [ ] Test error identity, typed nil errors, wrapping, and defer order.
- [ ] Define source positions for generated names and statements.
- [ ] Check how extension type inference uses `go/types` before final emission.
- [ ] Test getters against existing method expressions and inaccessible fields.
- [ ] Specify comprehension result typing; reject untyped nil in the first version.
- [ ] Test nil input, empty output, side effects, and changes to the backing array.
- [ ] Compare error-branch review time with ordinary Go on the same task.
- [ ] Reject union export until a Go boundary design preserves the claimed checks.

## Source findings

- [Go specification](https://go.dev/ref/spec): interfaces have a nil zero value.
  Non-basic interfaces cannot be ordinary value types. Package privacy cannot
  prevent zero initialization. These are limits on the design, not bugs in generation.
- [Union proposal](https://github.com/golang/go/issues/57644): prior design for sum
  types through general interfaces. It is evidence to study, not a compiler backend.
- [Go parser](https://pkg.go.dev/go/parser): parses Go source. New grammar needs an
  extended parser before Go AST checks and emission.
- [Go generation](https://go.dev/blog/generate): generation is separate from building.
  The proposed constructor generator must run as an explicit build step.

Read on 2026-10-08. All extension examples are proposed syntax.

## Compiler plan

Use an extended parser and checks for new forms. `go/parser` alone cannot parse
the proposed syntax. Convert new forms to Go AST nodes; run Go type checks and
emit formatted `.go` files. Use `//line` directives for source error locations.

Build generated packages with `go build` and test them with `go test`. Keep output
separate from source. Use fresh internal names. Preserve imports, package paths,
build constraints, and ordinary Go declarations. Pin and test a Go version for each
release. Future Go versions need new compatibility checks.

```text
extended source -> parser and extension checks -> Go AST -> .go -> Go compiler
ordinary Go caller -> generated package -> ordinary Go dependency
```

Sugar must expose ordinary Go types. These calls share the Go runtime and garbage
collector; they need no C FFI. Existing C dependencies keep their normal cgo build.
New type rules need a separate boundary design; syntax translation cannot enforce
them on ordinary Go callers.


## Checks

1. Build the error-branch parser, emitter, and a small Go compatibility corpus.
2. Compare source and generated Go for wrapping, partial results, scope, and defers.
3. Add constructor generation as a separate tool. Test zero values and package access.
4. Add getters and comprehensions separately, with output and effect-order tests.
5. Keep unions gated until the generated representation and Go boundary pass review.

For each addition, compile and run equivalent plain Go and extended-source cases.
Test ordinary Go callers and dependencies. Reject invalid programs at source lines.
Run unchanged Go samples through the compiler; increase the corpus before a superset
claim. Include methods, interfaces, generics, build constraints, and package init.

Compare source size and review findings on the same small business tasks. Keep an
addition only when it removes repeated code without hiding error policy or states.
