// Package frontend represents TGo syntax for Go parsing and type checking.
//
// Ordinary Go syntax should keep ordinary Go meaning. The frontend records the
// added postfix operators separately, including their positions and wrapping
// choice. The private checker applies TGo error rules with Go type rules.
//
// A marker call identifies each operator in the syntax tree. The private
// checker gives that expression its success types. Lowering replaces the marker
// with the real call, error check, and return.
//
// The frontend preserves the source operation's identity. It does not choose
// where generated statements execute; that decision needs checked types and
// belongs to the lowerer.
package frontend
