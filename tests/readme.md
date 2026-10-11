# Compiler integration tests

This package tests the compiler through `pkg/transpiler`. It checks generated
Go, source diagnostics, and runtime behavior. Compiler unit tests remain beside
the implementation they test.

`TestPropagationFixtures` compiles inputs from `testdata/propagation/`, compares
the result with committed `_tgo.go` snapshots, and checks handwritten output.
Generated test files also run as Go tests. The source files drive compilation;
old snapshots do not supply input.

Expression proof tests insert neutral statements and early returns at recorded
sites. They compare traces, storage, panics, and deferred calls with handwritten
expectations. Separate Go reference programs check corresponding propagation
cases. Rejection fixtures require specified source diagnostics.

The tagged lowering fuzzer compares generated TGo programs with independent
Go references. Bytes select program forms, operand values, and failure points.

See the [test guide](../docs/guide/testing.md) for focused commands, snapshot
updates, fixture layout, and the expression inventory. See
[the fuzz guide](fuzzing.md) for the separate fuzz campaign.
