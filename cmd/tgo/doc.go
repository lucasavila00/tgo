// Package main builds the tgo command and connects it to the transpiler library.
//
// Build the executable from the repository root:
//
//	make prepare
//	go build ./cmd/tgo
//
// The command takes one directory argument and passes it to transpiler.Compile:
//
//	./tgo /path/to/module
//
// See the [user guide] to compile and run a project. Use the [transpiler package]
// to call the compiler from a Go program.
//
// [user guide]: https://github.com/lucasavila00/tgo/blob/main/docs/guide/transpiler.md
// [transpiler package]: https://github.com/lucasavila00/tgo/blob/main/pkg/transpiler/readme.md
package main
