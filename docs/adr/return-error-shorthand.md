# Short returns for values and errors

Issue: [#321](https://github.com/lucasavila00/tgo/issues/321)

## Context

Error returns repeat zero values. Success returns repeat nil.

## Decision

Add these return forms:

```text
return .. err       // replaces return zero, err
return value ..     // replaces return value, nil
return x, y ..      // replaces return x, y, nil
```

Leading `..` takes one error operand and supplies typed zero success results.
Success results precede the error. Trailing `..` supplies nil.
The final result must implement `error`; trailing `..` also requires nil to fit.
Explicit operands follow Go return assignment rules. A sole multiple-result
expression can fill the success results. Bare `return ..` is invalid.

Keep Go token boundaries; write `1 ..`, not `1..`. Trailing `..` ends a statement
at a newline.

## Consequences

Lower to explicit Go returns with fresh typed locals. Evaluate operands once
in Go's required order. Preserve named-result assignment before defers, enclosing
returns, and jumps. Return errors unchanged, including typed nil conversions.

Compare generated programs with independent Go references for values, effects,
error identity, defers, generic types, and multiple-result calls. Check invalid
syntax, incompatible types, shadowed names, and source positions.

Automatic fixes must prove omitted values and effects stay unchanged.
