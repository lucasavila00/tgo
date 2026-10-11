// Package controlflow checks which source paths can reach a branch value.
//
// An if expression can contain return, break, continue, or goto before its final
// expression. These paths leave the branch without supplying a value. The
// checker must exclude them when it chooses the result type. Lowering must use
// the same decision when it emits an assignment to the result variable.
//
// The analysis follows statements and expressions through a graph. Each point
// has edges to the next possible steps. Return ends a path. Loop exits, labels,
// and fallthrough have explicit targets. Calls into real function literals keep
// their own control flow. The checker supplies the meaning of added conditional
// calls, panic, and unevaluated operands. This package depends only on Go syntax.
package controlflow
