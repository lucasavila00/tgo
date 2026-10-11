// Package transpiler exposes TGo compilation and expression rewriting.
//
// Compile writes generated Go in the source module and returns file contents.
// The caller can also write those files elsewhere. Commands and other tools use
// the same compiler.
//
// Load and InsertAfter support tools that need statements at an expression's
// evaluation point. The callback receives evaluated results, not source text
// that could run the expression again. Source keeps its original checked syntax
// so callers can apply another rewrite.
//
// Go package types remain visible through Source.Package. Keep this API small;
// module policy and lowering details belong to internal packages.
package transpiler
