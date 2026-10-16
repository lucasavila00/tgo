// Package checker decides whether source is valid before code generation.
//
// It checks types, names, and jumps on the native source nodes. For example,
// fetch()! must have a final error result that the source function can return.
// An if expression checks each completing branch against the required type.
// A branch that returns from the function does not supply a value.
//
// Each expression records its lexical scope. Each jump records its source
// target. The lowerer uses these facts when it moves evaluations into Go
// statements. A comprehension checks its range source before adding that
// range's bindings; later clauses and output see those bindings.
//
// The loader shares private type objects across packages and test variants.
// Ordinary Go loading retains the caller's stock syntax and type objects.
// Private checking reads its own syntax, while generated output is checked
// with stock Go. See [checker maintenance] for the shared version pin and patches.
//
// [checker maintenance]: https://github.com/lucasavila00/tgo/blob/main/internal/checker/upstream.md
package checker
