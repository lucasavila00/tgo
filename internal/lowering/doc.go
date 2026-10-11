// Package lowering replaces checked TGo expressions with equivalent Go code.
//
// Lowering means turning an expression that Go cannot execute directly into
// ordinary Go statements. This package uses the types and name declarations
// from the [checker]. It returns formatted file contents; the [driver] writes
// those contents to disk.
//
// # Preserve when expressions run
//
// Consider this TGo call, where fetch returns a value and an error:
//
//	use(first(), fetch()!, last())
//
// The generated code must call first before fetch, and call fetch only once.
// If fetch fails, the function must return before last or use runs. The lowerer
// saves earlier results in local variables so inserted error checks do not
// change this order.
//
// Conditional execution must also stay conditional. In left() && right()!,
// right runs only if left returns true. In a loop condition, an inserted check
// must run each time the condition runs. A continue must still execute the
// loop's post statement before the next condition check.
//
// Assignment needs the same care. The generated code first evaluates the
// destination expressions and the values, then performs the stores in order.
// It must not move a possible store panic before an earlier error return.
//
// # Keep statements in their source function
//
// An if expression becomes branches that assign to a result variable. For
// example, this source:
//
//	name := if ready { "ready" } else { "waiting" }
//
// has the behavior of:
//
//	var result string
//	if ready {
//	    result = "ready"
//	} else {
//	    result = "waiting"
//	}
//	name := result
//
// A branch that returns or jumps out needs no result assignment. A comprehension
// becomes range loops that add values to a result slice or map. Inside a source
// function, these rewrites keep statements in that function. Returns, deferred
// calls, and jumps must retain their original function and targets.
//
// # Preserve types and errors
//
// Saved values use their checked types. For example, saving 1 << n as an int
// can change a result that the source computes as uint64. Generated names and
// import aliases must also avoid names already used by the source.
//
// Check an error before converting it to the enclosing function's result type.
// A nil pointer converted to an error interface becomes a non-nil interface.
// Checking after that conversion would incorrectly treat success as failure.
// An error interface that already holds a nil pointer is non-nil and is a failure.
//
// The ! operator returns the original error. The !! operator adds the call name
// with Go's %w error wrapping, so errors.Is and errors.As can find the original
// error. Other function results receive zero values of their declared types.
//
// The ordinary-Go InsertAfter API uses the same statement generation rules.
// See the [public API] for an example. Rewrites leave the loaded syntax unchanged,
// so callers can produce independent changes from one load. The [runtime tests]
// compare generated behavior with handwritten Go, including order and jumps.
//
// [checker]: https://github.com/lucasavila00/tgo/blob/main/internal/checker/readme.md
// [driver]: https://github.com/lucasavila00/tgo/blob/main/internal/driver/readme.md
// [public API]: https://github.com/lucasavila00/tgo/blob/main/pkg/transpiler/readme.md
// [runtime tests]: https://github.com/lucasavila00/tgo/blob/main/tests/readme.md
package lowering
