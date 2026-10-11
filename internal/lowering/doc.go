// Package lowering preserves expression behavior when adding statements.
//
// An expression can run after earlier operands, inside a selected branch, or
// on each loop visit. Added statements must run at that same point. For
// use(first(), target()!, last()), first runs before target; failure returns
// before last runs. Moving every check to the start of the statement would
// change that behavior.
//
// Checked types determine temporary storage. Explicit types preserve conversions
// that the surrounding source supplies. Fresh names preserve source bindings.
// Assignments retain their original storage, and each function retains its own
// return target.
//
// Rewrites copy changed syntax. The checked input stays available for another
// rewrite. Module selection and file output belong to the caller.
package lowering
