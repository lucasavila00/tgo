// Package checker validates TGo types, scope, and jumps before lowering.
//
// Writing source at a Go path lets Go select files and imports. It does not
// teach Go the meaning of added expressions. Loader reads those files through
// the [frontend], then applies TGo rules with a private copy of Go's checker.
//
// # Propagation
//
// If fetch returns (Value, error), fetch()! supplies Value to its surrounding
// expression. For outer(inner()!)!, check the inner success type before the
// outer call, within one source check. There is no package reload per operator.
//
// Several success results stay separate; an error-only call leaves no value.
// Check the final error result and the enclosing function's error interface.
// The original error must fit that return for !; wrapped error must fit it for !!.
// Actual error checks and returns are emitted by [lowering].
//
// # If values
//
// Check each completing branch against its contextual type. Without context,
// use branch types and Go constant rules. Nil alone supplies no type. In a
// generic call, other arguments can supply the type:
//
//	pair(if ok { nil } else { nil }, pointer)
//
// For pair[T any](a, b T) and pointer of type *int, T is *int. Branches that
// return or jump out supply no inference value. [Control flow] identifies
// reachable branch endings. Branch returns, defers, and jumps remain in their
// enclosing source function; frontend placeholders do not create a new one.
//
// # Packages and maintenance
//
// Check source dependencies as needed and cache results by package ID. Test
// variants have separate IDs. Read external dependency export data with the
// matching private importer. Use the selected Go architecture's word sizes.
//
// Keep private type objects separate from stock objects. Public Load retains
// its stock package and checks the same ordinary syntax separately for lowering.
// The [driver] checks final output with stock Go after all source checks pass.
//
// Run make prepare to generate the checker and importer from the same pinned
// Go version and tracked patch. See [preparation] for Go updates, [the decision]
// for the tradeoff, and [measurements] for recorded performance.
//
// [frontend]: https://github.com/lucasavila00/tgo/blob/main/internal/frontend/readme.md
// [lowering]: https://github.com/lucasavila00/tgo/blob/main/internal/lowering/readme.md
// [Control flow]: https://github.com/lucasavila00/tgo/blob/main/internal/controlflow/readme.md
// [driver]: https://github.com/lucasavila00/tgo/blob/main/internal/driver/readme.md
// [preparation]: https://github.com/lucasavila00/tgo/blob/main/internal/checker/upstream.md
// [the decision]: https://github.com/lucasavila00/tgo/blob/main/docs/adr/source-checking.md
// [measurements]: https://github.com/lucasavila00/tgo/blob/main/docs/performance.md
package checker
