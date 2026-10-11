# HOW: lower expressions into ordinary Go statements

Issue: [#248](https://github.com/lucasavila00/tgo/issues/248)

Contract: [WHAT](expression-statement-insertion.md)

## Terms

- **Lowering:** replace source constructs with simpler constructs.
- **Temporary variable:** a generated local that saves an evaluated value.
- **Assignment target:** a variable, field, index, or dereference to be written.
- **Statement list:** an ordered list of `go/ast` statement nodes.

## Proposed algorithm

- Adapt the approach of [Go's ordering pass][order] to `go/ast` and `go/types`.
  Build new AST nodes from unchanged, type-checked input. No additional
  intermediate language, runtime wrapper, or state machine.
- Lower each expression into a statement list and replacement expressions.
  Carry source bindings, expected types, constants, and result counts.
- The enclosing construct places that list where the expression executes.
  For a conditional operand, this is inside its branch, not before the entire
  source statement.
- Before emitting a later operand's statements, save earlier pending
  evaluations in temporaries. Rebuild the parent from those saved values.
  Keep untyped constants in their original conversion context. Preserve
  storage for implicit pointer receivers; save the method value or receiver
  address, not a copied receiver.
- Insert before-return at expression entry, before its operands. Insert
  after-return after its complete evaluation. Retain unreachable syntax that
  keeps source variables used and labels available.

Example: insertion after `right()` in `use(left() && right())`:

```go
r := left()
if r {
    r = right()
    return err
}
use(r)
```

## Placement rules

- **Boolean expressions:** use the source expression type for result
  temporaries. For a constant left operand, lower only the evaluated branch;
  keep a skipped expression in its original typing context.

- **Declarations, including `:=`:** emit each declaration and its temporaries
  in the original block. Process grouped declarations one specification at a
  time. Move a source label to the first emitted statement, before temporary
  declarations. Later statements retain access to the source names.
- **Other statements:** put temporaries and the rewritten statement in a new
  block. Put an existing source label on that block, so `goto` enters before
  its declarations. If a wrapper changes a labeled break/continue target,
  map that jump to the enclosed loop, switch, or select label.
- **Assignment:** save index operands and pointer values, evaluate RHS values,
  then store left to right. Preserve array storage. Do not prepare a target by
  taking `&array[index]` if that introduces an earlier bounds panic.
- **Loops:** keep the native `for` initializer and per-iteration variables.
  Move post evaluation to the next iteration's body entry; skip it on the
  first iteration with a generated boolean. Then evaluate the condition and
  execute the source body in a nested block. This keeps the native variable
  copy before post evaluation. `continue` reaches post; `break` skips it.
  Map labeled break/continue to the generated loop label; keep the source
  `goto` label on the enclosing block.
- **Switches:** emit ordered, correctly typed case comparisons and labeled
  case bodies. Translate fallthrough into a jump to the next body. Keep an
  enclosing switch for break. Keep type switches and their bindings native.
- **Ranges:** keep native range and its skipped-expression rules. Preserve
  native `:=` iteration bindings. For assignment targets, range into temporary
  variables and perform target evaluation inside the body.
- **Selects:** lower operands before selection. Receive into temporaries;
  lower targets inside the selected body. After-receive insertion precedes
  target evaluation. Native Go cannot insert a statement between selection and
  communication; report that insertion boundary as unrepresentable.
- **Go/defer:** lower operands before the native scheduling statement. An
  after-root insertion follows scheduling or defer registration; it does not
  execute the scheduled call.
- **Nested functions:** use separate lowering passes and return targets.

## Verification

- Prove each rewrite against WHAT before claiming complete support.
- Type-check generated Go and compare runtime traces; then format it.

[order]: https://go.dev/src/cmd/compile/internal/walk/order.go
