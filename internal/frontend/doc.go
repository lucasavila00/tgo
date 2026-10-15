// Package frontend reads TGo source into native syntax nodes.
//
// The parser reads the complete source once. A postfix operator belongs to its
// actual operand. An if expression owns its branches. A comprehension owns its
// written outputs and clauses. The checker and lowerer use these same nodes,
// so types, bindings, and error positions refer to the source that the user wrote.
//
// Value branches and collection clauses belong to the enclosing source
// function. Returns, defers, and jumps keep that function's meaning. A function
// literal appears in the tree only when the source contains a function literal.
//
// Physical file names select Go build inputs. Logical source names and line
// directives identify diagnostics. Both use the original input positions.
// The frontend also saves source call text for the context added by !!.
package frontend
