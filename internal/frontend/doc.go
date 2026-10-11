// Package frontend represents TGo syntax for Go parsing and type checking.
//
// Ordinary Go syntax should keep ordinary Go meaning. The frontend records the
// added postfix operators separately, including their positions and wrapping
// choice. This lets the driver check TGo error rules while Go checks the
// surrounding expression.
//
// A checking helper exposes a call's success results. It is a description for
// the type checker, not an error-handling implementation. Lowering must replace
// it with the real call, error check, and return.
//
// The frontend preserves the source operation's identity. It does not choose
// where generated statements execute; that decision needs checked types and
// belongs to the lowerer.
package frontend
