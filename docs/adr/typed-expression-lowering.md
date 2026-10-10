# Replace prefix-list rewriting with a typed lowering plan

Issue: [#245](https://github.com/lucasavila00/tgo/issues/245)

## What is wrong today

The compiler lowers expressions while it changes the Go AST in place.
`expression()` returns a replacement expression and a flat list of statements.
The caller must find a safe place for that list. The list does not say whether
its work runs once, on each loop test, or only after a branch is selected.

Generated call results also lose their types at this boundary. Later helpers
query old type maps or recover some types from variable names and AST shapes.
Adding more statement handlers can fix individual cases, but does not fix
these contracts. The current model is too weak for general expression lowering.
This is a design limit, not evidence that valid TGo cannot be compiled.

## Proposed change

Replace direct AST rewriting with two separate passes: build a typed lowering
plan for each function, then emit Go AST from that plan. Expression lowering
produces operations and typed value IDs, not loose Go statement prefixes.

The plan is a structured tree with these records:

```text
Value: ID, type, source position
Place: ID, target type, evaluated target operands
Block: ordered operations, source scope
Operation: evaluate, bind, store, branch, loop, switch, select, return, jump
Target: source label or loop/switch/select target identity
```

Branch, loop, switch, and select operations own child blocks. A loop has
initializer, test, body, and post blocks. A select has entry operand evaluation
and selected-case blocks. These boundaries state when work executes.

One recursive expression lowerer writes into a specified block and returns
typed values or places. It receives contextual types and required result counts.
Propagation produces a call with typed result IDs and an error branch.
Comprehensions produce loops. Both use this path in every expression context.

For `ready && check()!!`, the call and error return belong inside the true
branch of `ready`. In a loop condition they belong in the test block.
No caller receives a prefix that it must position by hand.

Statement lowering defines execution regions once for each Go statement kind.
It does not inspect expressions to implement propagation or comprehensions.
New expression forms use the shared path; new statement forms must define their
execution rules. Scope and jump target identities remain attached to the plan.

Only the emitter turns plan values into Go identifiers and operations into Go
statements. It can retain unchanged Go subtrees with captured semantic facts.
Generated values do not use the old AST type maps or name-based type recovery.
The Go type checker and formatter remain dependencies, not the decision.

## Required behavior and limits

The emitter must preserve Go iteration bindings, post timing, switch case
order, and select entry versus selected-case evaluation. It must also preserve
assignment operand evaluation before stores, array addressability, map targets,
receiver capture, contextual types, scopes, and jumps. The plan makes these
rules explicit; it does not implement them by itself.

This is a compiler refactor, not a language change. Do not use function wrappers
or reject valid code to simplify it. `!!` returns the same error interface;
`!` wraps it once with the specified call name.

## Research and implementation plan

The [Go ordering pass][go-order] uses typed locals and expression-owned work,
but still implements statement-specific execution rules. [Rust MIR][rust-mir]
makes destinations and control flow explicit. Use these principles in a
structured plan, not Rust's full graph and destruction machinery.

After approval, build the plan and emitter, then replace the current lowering
path. Keep runtime regressions and add nested-expression tests across all
statement contexts. Verify order, skipped work, types, scopes, captures, and
jumps in hosted CI. Remove replaced helpers and name-based type recovery; keep
no fallback. Move the contract to the compiler guide and delete this ADR.

[go-order]: https://go.dev/src/cmd/compile/internal/walk/order.go
[rust-mir]: https://rustc-dev-guide.rust-lang.org/mir/construction.html
