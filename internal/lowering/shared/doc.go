// Package shared supplies the operations used by TGo lowering rules.
//
// An expression can need statements before its value is available. Result keeps
// those statements with typed values until their caller places them in Go syntax.
// Value retains the checked type and source identity, including tuple results.
// Operand evaluation saves earlier values before adding later statements.
// Assignments evaluate destinations without reading them, then store in source
// order. These operations give all rules the same execution and type behavior.
//
// Context holds state for the current function. New function and initializer
// contexts share request names and file imports, but have separate returns,
// labels, jumps, and temporary values. Source inventory stays unchanged.
// Expression and statement callbacks receive the active context explicitly.
// They return to the lowering dispatcher when an operand contains another rule;
// this package has no dependency on rule packages.
//
// Ordinary and value switches share case selection and jump handling. A value
// switch supplies a ValueBlock callback as a local parameter to transform its
// branch results. The switch passes its current context to that callback.
// Ordinary switches do not supply a transformer. This parameter keeps the
// value-expression rule separate without duplicating switch control flow.
package shared
