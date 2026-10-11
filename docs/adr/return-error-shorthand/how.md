# Short returns: HOW

## Approach

Recognize adjacent `..` only before or after return operands. Keep Go token
boundaries: write `1 ..`, not `1..`. Trailing `..` ends statements at
newlines. Leading `..` needs an operand, which can start on the next line.

The frontend records the operator and original positions on a return
statement. Prepare a marker, an internal node without a runtime call, for
Go's parser. The checker uses enclosing result types, contextual constants,
and generic inference. Count the statement as a return for termination
checks.

## Example

Reuse propagation's fresh typed zero locals for `return .. err`:

```go
var zero T
return zero, err
```

For `return pair() ..`, save results before appending nil:

```go
first, second := pair()
return first, second, nil
```

Use checked type identities and fresh names. If source code shadows `nil`,
use zero storage of the final type. Existing expression lowering preserves
operand order, enclosing returns, defers, and jumps.

## Tradeoffs

Library helpers need type arguments for omitted results and separate forms
for result counts. Return syntax uses the enclosing signature. Automatic
fixes must prove omitted values and effects stay unchanged.

[WHAT](what.md) states behavior. [PROOF](proof.md) defines checks.
