// Package frontend reads TGo source for the type checker.
//
// Go's parser cannot read TGo operators such as fetch()! or an if expression.
// The frontend temporarily replaces added syntax with forms that Go's standard
// parser can read. It returns a syntax tree: a structure that describes the
// expressions, statements, and declarations in the file.
//
// The result also records which parts of that tree represent TGo operations.
// These records let the [checker] apply TGo rules instead of treating the
// replacement forms as ordinary Go calls. The frontend does not decide types
// or generate the final Go code.
//
// # Preserve the meaning of source positions and statements
//
// Temporary text can be longer than the original expression. The frontend maps
// positions back to the input so errors refer to the TGo text that the user
// wrote. The physical Go filename remains available for Go's file selection.
//
// Some replacement forms contain function literals to hold branch statements.
// These forms do not introduce a function in the source program. For example,
// a return inside an if expression still exits the enclosing source function.
// The checker and [lowering] must preserve that rule.
//
// A comprehension builds a slice or map from range loops and optional filters.
// Its records preserve the order of those clauses and the output expression.
// This lets later compiler steps check names and types in the same scope that
// the source program defines.
//
// [checker]: https://github.com/lucasavila00/tgo/blob/main/internal/checker/readme.md
// [lowering]: https://github.com/lucasavila00/tgo/blob/main/internal/lowering/readme.md
package frontend
