// Package frontend reads TGo source for the compiler.
//
// The private Go parser reads every supported TGo form in one parse. It
// records the exact source nodes for error operators, value expressions,
// comprehensions, and short returns. Grammar rules decide which call an
// operator belongs to and which braces belong to an expression.
//
// # Keep source meaning before building checker calls
//
// After parsing succeeds, the frontend reserves source identifiers and saves
// source call names. It then names the recorded syntax containers and builds
// calls for checking. A single bottom-up walk inserts postfix calls, so a
// nested generated helper cannot enter another call's error context.
//
// Conditional branches and comprehension clauses remain statements in the
// enclosing source function. Their syntax containers let checking and lowering
// apply TGo rules without changing return targets or source bindings.
//
// Physical file names identify files for Go build selection. Logical source
// names and line directives identify positions in diagnostics. The scanner
// records both from the original input. ParseFile returns the completed syntax
// and source records together.
package frontend
