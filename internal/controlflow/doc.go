// Package controlflow finds which paths can reach an if branch's value.
//
// An if value can contain statements before its final expression. A return,
// break, continue, or goto can leave the branch before that value is reached.
// The checker must know which paths supply a value to determine its type. The
// lowerer needs the same answer before it emits a result assignment.
//
// Branch reports whether execution can reach the last statement and whether it
// can reach the end of the branch. The analysis builds a graph of possible next
// steps. Returns end paths. Loops, labels, and switch fallthrough have jump
// targets. Real function literals have separate control flow.
//
// The caller supplies functions that identify added if expressions, terminating
// calls such as panic, and unevaluated operands. This package reads Go syntax;
// it does not type check source or generate code.
package controlflow
