// Package transpiler exposes module compilation and expression insertion.
//
// Compile takes a directory inside a Go module. It writes generated Go beside
// owned TGo inputs and checks the final module. It returns nothing. Invalid
// source and compiler failures panic. Original TGo and ordinary Go source remain
// on disk.
//
// Load and InsertAfter are a separate API for tools that add statements after an
// ordinary Go expression. Load reads one package and records expression sites.
// InsertAfter supplies saved expression results to the caller's callback and
// returns formatted Go file contents without writing them.
//
// Source.Package retains stock Go package and type objects. Private checking
// provides the lowerer's types separately. The checked tree stays unchanged, so
// one load can support several independent rewrites.
package transpiler
