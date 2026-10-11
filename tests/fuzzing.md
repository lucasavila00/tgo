# Lowering fuzz tests

Run from the repository root:

```sh
GOMAXPROCS=2 go test -tags=loweringfuzz ./tests -run '^$' \
  -fuzz '^FuzzPropagationBehavior$' -parallel=1 -fuzztime=10m
```

The `loweringfuzz` build tag keeps this campaign out of the normal test suite.
No CI change is needed. Go saves a failing input in
`tests/testdata/fuzz/FuzzPropagationBehavior/` and prints a command to replay it.

## What the test checks

The generator makes bounded TGo programs with calls, assignments, array and
slice stores, maps, short-circuit expressions, loops, switches, selects,
deferred arguments, method calls, and operations that can panic. Bytes choose
the context, operand values, failing call, and propagation operator.

Each input has a separate Go reference. The reference uses a helper that stops
the enclosing function on an error. It uses a private panic value and catches
only that value at the function boundary. Ordinary panics still reach the
program's outer panic handler. Source deferred calls run before the boundary
handler. The generated programs do not use `recover`, goroutines, or nested
functions that propagate errors.

The test compares call traces, values, final storage, error text, error identity,
error chains, and panic types. The reference does not use TGo or compiler
lowering code. Both programs run with the current Go toolchain.

## How to use a failure

Replay the saved input. Read the TGo source, Go reference, and generated output
in the failure report. Reduce the input and confirm that Go specifies the order
on which the failure depends. Go does not specify the order of every variable
read relative to a call. A difference in an unspecified order is not sufficient
evidence of a compiler bug.

Open a bug issue only after that check. Include the small input, actual result,
expected result, and compiler version. Keep the saved input so the failure can
be replayed. Add a normal regression fixture with the fix.

This generator checks a bounded set of program forms. It does not prove that
all valid TGo programs compile correctly. The separate audits check types,
source positions, build selection, closures, and other forms.
