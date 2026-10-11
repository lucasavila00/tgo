// Package transpiler exposes TGo compilation and expression rewriting.
//
// The public API separates translation from file output. Compile returns file
// contents; the caller chooses where to write them. This lets commands and other
// tools use the same compiler.
//
// Load and InsertAfter support tools that need statements at an expression's
// evaluation point. The callback receives evaluated results, not source text
// that could run the expression again. Source keeps its original checked syntax
// so callers can apply another rewrite.
//
// Go package types remain visible through Source.Package. Keep this API small;
// module policy and lowering details belong to internal packages.
package transpiler
