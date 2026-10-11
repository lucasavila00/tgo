// Package lowering replaces checked expressions with equivalent Go statements.
//
// The [checker] supplies syntax, types, bindings, and operation records.
// LowerPropagations handles propagation and if values. It returns formatted
// file contents; the [driver] writes them to owned generated paths.
//
// # Calls and branches
//
// For use(first(), target()!, last()), save first before calling target once.
// If target fails, return before last or use runs. Preserve the order Go defines;
// do not impose order where Go leaves it unspecified. For left() && right()!,
// evaluate right and check its error only on the true path of left.
//
// An if value uses a result local and ordinary branches:
//
//	var chosen string
//	if ok {
//	    log("yes")
//	    chosen = "yes"
//	} else {
//	    chosen = "no"
//	}
//	name := chosen
//
// A branch that returns or jumps out needs no result assignment. Returns,
// defers, and jumps keep their source function and targets.
//
// # Errors and saved types
//
// Check the original error before return conversion. A nil concrete pointer
// can become a non-nil error interface if converted first. An error interface
// that already contains a nil pointer is failure. Generic error checks use
// reflection when direct nil comparison is unavailable.
//
// ! returns the same error; !! adds call-name context with %w in the failure
// branch. Other results get typed zero values. Result names and source defers
// retain their meaning.
//
// Save values with checked types and fresh names. For example, var saved uint64
// = 1 << n retains its source conversion; inferred int could change the result.
// Print types by object identity. Required types must be accessible in the
// generated scope. Fresh import aliases avoid source names.
//
// # Loops and storage
//
// Keep conditions at each loop visit. Run post work after the first visit;
// continue must still reach it. An if value in a loop initializer can break or
// continue that loop. Preserve separate := iteration variables and closure
// bindings when generated control flow requires renaming.
//
// In selects, evaluate channel and send operands before selection. Evaluate
// receive-assignment destinations only inside the selected case.
//
// Record assignment destinations without reading them, evaluate right-hand
// values, then store. Keep the original array or evaluated slice and map. A
// later store panic must not move ahead of an earlier propagation return.
//
// InsertAfter is a separate ordinary-Go API; see the [public API] for use.
// Both operations copy changed syntax without modifying checked input, so one
// load can support independent rewrites. [Tests] compare execution with
// handwritten references; valid Go alone does not prove equivalent behavior.
//
// [checker]: https://github.com/lucasavila00/tgo/blob/main/internal/checker/readme.md
// [driver]: https://github.com/lucasavila00/tgo/blob/main/internal/driver/readme.md
// [public API]: https://github.com/lucasavila00/tgo/blob/main/pkg/transpiler/readme.md
// [Tests]: https://github.com/lucasavila00/tgo/blob/main/tests/readme.md
package lowering
