// Package transpiler compiles TGo modules.
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
// [user guide]: https://github.com/lucasavila00/tgo/blob/main/docs/guide/transpiler.md
// [driver]: https://github.com/lucasavila00/tgo/blob/main/internal/driver/readme.md
package transpiler
