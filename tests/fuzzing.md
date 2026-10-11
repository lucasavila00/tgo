# Fuzz lowering against separate Go references

The `loweringfuzz` build tag enables two campaigns. Each mutation produces a
TGo program and a separate ordinary Go reference. The test compiles TGo in its
test directory, runs both programs, and compares their reported behavior.
References do not use compiler lowering code.

## Propagation

```sh
GOMAXPROCS=2 go test -tags=loweringfuzz ./tests -run '^$' \
  -fuzz '^FuzzPropagationBehavior$' -parallel=1 -fuzztime=10m
```

Bytes choose the context, operand values, failing call, and propagation operator.
Cases include calls, assignments, array and slice stores, maps, skipped Boolean
operands, loops, switches, selects, deferred arguments, methods, and panics.

The Go reference uses a private panic value to stop a function on propagation,
and catches only that value at the function boundary. Ordinary panics still
reach the outer reporting handler. Source defers run before the boundary handler.
Generated test bodies exclude source recover, goroutines, and nested propagating
functions; those would need different reference rules.

The comparison includes traces, values, final storage, error text and identity,
error chains, and panic types. Failures are saved under
`tests/testdata/fuzz/FuzzPropagationBehavior/`.

## If values

```sh
GOMAXPROCS=2 go test -tags=loweringfuzz ./tests -run '^$' \
  -fuzz '^FuzzIfExpressionBehavior$' -parallel=1 -fuzztime=10m \
  -fuzzminimizetime=1x -timeout=15m
```

Bytes choose statement contexts, nested branches, conditions, values, and
propagation failures. The comparison includes traces, results, deferred calls,
errors, and error identity.

Value-only reference branches use Go functions. References for return, defer,
break, continue, and goto use ordinary statements so those operations retain
their function and loop targets. Cases also cover generic inference, contextual
types, loops, skipped operands, ranges, and selects. Failures are saved under
`tests/testdata/fuzz/FuzzIfExpressionBehavior/`.

Each mutation compiles and runs two programs. The command limits automatic
reduction to one execution per input; reduce confirmed failures separately.

## Investigate a failure

Go prints a replay command for a saved input. Replay it, read the source,
reference, and generated Go in the report, then reduce the case. Check that Go
specifies the evaluation order on which the difference depends. A difference
in unspecified order is not sufficient evidence of a compiler bug.

A confirmed report should include the small input, expected and actual result,
and compiler version. Keep the saved input and add a normal regression fixture
with the fix.

These campaigns cover bounded program forms, not every valid program. Normal
CI does not run them. The normal tests separately check types, positions, build
selection, closures, and other cases.
