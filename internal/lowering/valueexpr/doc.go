// Package valueexpr lowers if and switch expressions into branches and typed storage.
//
// A value expression produces a value from the branch it reaches. The rule
// declares a result with its checked type and stores each completing branch's
// final expression into that result. A branch that returns or jumps out does not
// store a result. Branch effects stay conditional, and statements remain in
// the source function rather than moving into a generated function.
//
// Shared operations evaluate conditions and switch tags, allocate names, and
// preserve assignments and jumps. Value switches use the same case selection
// and control flow as ordinary switches. This rule passes its branch transformer
// to shared switch lowering as a local callback, with the current context passed
// explicitly when the switch invokes it.
//
// A branch can contain propagation or a comprehension. Recursive expression and
// statement evaluation goes through the dispatcher with the current context.
// The package boundary keeps branch-result decisions here while shared code
// retains the same type, evaluation, and control-flow rules for all features.
package valueexpr
