// Package transpiler compiles TGo modules and adds statements to ordinary Go code.
//
// # Compile a TGo module
//
// Call Compile with a directory inside a Go module:
//
//	transpiler.Compile("/path/to/module")
//
// Compile processes the whole module and writes Go files beside the TGo inputs.
// For example, service.tgo produces service_tgo.go. It checks the final module
// with Go. It returns nothing; invalid source or a compiler failure causes a
// panic. See the [user guide] for TGo syntax and the [driver] for file handling
// and failure behavior.
//
// # Add code after an expression runs
//
// Load and InsertAfter let a tool change ordinary Go source. For example, a
// tool can print the value from fetch before consume uses it. Given:
//
//	consume(fetch())
//
// where fetch returns int, the changed code has this behavior:
//
//	var saved int = fetch()
//	println(saved)
//	consume(saved)
//
// The saved value prevents a second call to fetch. The added code runs only
// when the selected expression runs. For example, if the expression is in a
// branch that is skipped, the added code is skipped too.
//
// Use Load to read and type-check one Go package. Then select the expression
// from Source.Sites. Each Site identifies an expression in the original file;
// a nonempty Reason explains why insertion is not available for that expression.
//
// InsertAfter calls a function that you supply to build the added statements.
// That function receives Go syntax nodes for the expression's saved results,
// in result order. Use these nodes to refer to the values without evaluating
// the original expression again.
//
// In this example, source is the result from Load, and site selects fetch().
// The callback builds a println statement with the saved value as its argument.
// It uses the go/ast package to describe the new Go code:
//
//	files, err := source.InsertAfter(site, func(results []ast.Expr) []ast.Stmt {
//	    return []ast.Stmt{&ast.ExprStmt{X: &ast.CallExpr{
//	        Fun:  ast.NewIdent("println"),
//	        Args: results,
//	    }}}
//	})
//
// Check err before using files. This example assumes that fetch returns one
// value and that println names Go's built-in function.
//
// The returned map contains formatted Go files keyed by absolute path.
// InsertAfter does not write them; the caller decides which files to save.
// It also leaves the loaded syntax unchanged. Each call makes an independent
// rewrite, so changes from separate calls do not accumulate.
//
// [user guide]: https://github.com/lucasavila00/tgo/blob/main/docs/guide/transpiler.md
// [driver]: https://github.com/lucasavila00/tgo/blob/main/internal/driver/readme.md
package transpiler
