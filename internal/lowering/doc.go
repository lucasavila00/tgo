// Package lowering replaces checked expressions with equivalent Go statements.
//
// The driver supplies source syntax, checked types, bindings, and TGo operation
// records. LowerPropagations handles propagation and if values. It returns
// formatted Go file contents; the driver writes owned generated files.
//
// The lowerer evaluates each call once, checks its error at that point, and
// returns on failure before later operands run. It saves earlier operands when
// needed. Branch work stays in the selected branch, loop work keeps its repeated
// execution, and assignment destinations keep their original storage.
//
// Saved values use explicit checked types and fresh local names. Propagation
// checks the original error result before return conversion. Returns, defers,
// and jumps retain their source function and targets. If branches that leave the
// function or jump out need no result assignment.
//
// InsertAfter is a separate operation for ordinary Go. It inserts a caller's
// statements after one selected expression. Both operations build changed syntax
// without changing the checked input, so callers can make several rewrites from
// one source. Module selection and disk writes belong to the driver or caller.
package lowering
