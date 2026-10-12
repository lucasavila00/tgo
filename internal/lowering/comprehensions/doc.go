// Package comprehensions lowers collection expressions into Go ranges and stores.
//
// A comprehension produces a slice or map by evaluating its generators and
// adding their outputs. This rule owns collection construction, range selection,
// and output insertion. A slice with one output on every visit can allocate once
// when the checked source permits an exact count. Other forms append as their
// outputs run. Hidden builtin names use the existing shared import operations.
//
// Shared operations retain checked element and key types, give generator
// bindings fresh names, and evaluate output operands in source order. Their
// ordinary range and jump handling keeps filters, nested ranges, returns, and
// continues at the source execution point. Collection output state belongs to
// the current function context; nested functions do not inherit that state.
//
// Output expressions can contain other TGo rules. The supplied context returns
// to the dispatcher for those expressions. This keeps collection decisions
// separate from error and branch rules without changing their execution order.
package comprehensions
