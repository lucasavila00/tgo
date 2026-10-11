// Package transpiler exposes module compilation and expression insertion.
//
// Compile takes a directory inside a Go module, writes generated Go beside
// owned TGo inputs, and checks the final module. It returns nothing and panics
// on invalid source or compiler failure. See the [user guide] for language and
// commands, and the [driver] for the compilation process and shared terms.
//
// # Insert statements after a Go expression
//
// Load and InsertAfter are a separate API for ordinary Go. Load reads one
// package and records expression sites. Source.Package retains stock Go package
// and type objects. Private checking supplies lowering information separately.
//
// Source.Sites identifies expressions by file, byte offsets, and AST kind.
// Reason marks sites without value evaluation, such as type names. ErrorReturn
// identifies a surrounding function whose sole result is error.
//
// InsertAfter passes saved result references to a callback, then returns
// formatted file contents keyed by absolute paths. It does not write files.
// For a site in a function returning only error, with err in scope:
//
//	files, err := source.InsertAfter(site, func(results []ast.Expr) []ast.Stmt {
//	    return []ast.Stmt{
//	        &ast.ReturnStmt{Results: []ast.Expr{ast.NewIdent("err")}},
//	    }
//	})
//
// The caller handles err and writes files. Saved references do not reevaluate
// the original expression. Added statements run after it and before its parent
// uses the result, only when the expression executes. Checked syntax stays
// unchanged, so one load supports several independent rewrites.
//
// Saved values use explicit types. If a required type cannot be named in the
// generated scope, the error identifies its type and source position.
// An accessible type or conversion at that source boundary can resolve it.
//
// [user guide]: https://github.com/lucasavila00/tgo/blob/main/docs/guide/transpiler.md
// [driver]: https://github.com/lucasavila00/tgo/blob/main/internal/driver/readme.md
package transpiler
