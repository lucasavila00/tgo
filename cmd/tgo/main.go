package main

import (
	"flag"

	"github.com/lucasavila00/tgo/pkg/transpiler"
)

// main reads the source-directory argument and calls the public compiler API.
func main() {
	flag.Parse()
	if flag.NArg() != 1 {
		panic("usage: tgo source-directory")
	}
	transpiler.Compile(flag.Arg(0))
}
