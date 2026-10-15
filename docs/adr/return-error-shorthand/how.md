# Short returns: HOW

## Approach

Read Go's `ELLIPSIS` token as the first or last return-list element. Require
a comma between the placeholder and supplied operands. A trailing `...`
ends the statement at a newline. A leading `...` needs an error operand.

The frontend records return operands, the placeholder position, and omitted
results. The checker uses the enclosing signature: reject named results, including `_`, and check supplied
values against result types. Count the statement as a return for termination checks.

## Example

Reuse propagation's typed zero locals for `return ..., err`:

```go
var zero T
return zero, err
```

For `return pair(), ...` in `(int, string, error)`, use declared result types:

```go
var first int
var second string
first, second = pair()
return first, second, nil
```

Thus private callee types need no generated name. Use checked type identities
and fresh names. If source code shadows `nil`, use zero storage of the final
type. Existing expression lowering preserves operand order, returns, defers,
and jumps.

## Tradeoffs

Library helpers need explicit type arguments for omitted results and separate
forms for result counts. Return syntax uses the enclosing signature. Automatic fixes require
unchanged values and effects.

[WHAT](what.md) states behavior. [PROOF](proof.md) defines checks.
