# Lower short returns directly to Go

Issue: [#321](https://github.com/lucasavila00/tgo/issues/321)

Recognize adjacent `..` only before or after return operands. Keep Go token
boundaries: write `1 ..`, not `1..`. Trailing `..` ends a statement at newlines.
Leading `..` needs an operand, which can start on the next line.

The frontend records the operator and original positions on a return statement.
Prepare a marker, an internal node without a runtime call, so Go's parser can
read operands. The checker uses enclosing result types, contextual constants,
and generic
inference. Count the statement as a return for termination checks.

Emit explicit Go returns. Reuse propagation's fresh typed zero locals:

```go
var zero T
return zero, err
```

For a sole multiple-result operand, save its results before appending nil:

```go
first, second := pair()
return first, second, nil
```

Use checked type identities and fresh names. If source code shadows `nil`, use
zero storage of the final type. Existing expression lowering preserves operand
order, enclosing returns, defers, and jumps without helper functions.

Automatic fixes must prove removed operands have no effects and equal the
omitted destination values. Leave uncertain cases unchanged.

[WHAT](what.md) states behavior. [PROOF](proof.md) defines checks.
