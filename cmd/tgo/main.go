package main

import (
	"flag"

	"github.com/lucasavila00/tgo/pkg/transpiler"
)

func main() {
	flag.Parse()
	if flag.NArg() != 1 {
		panic("usage: tgo source-directory")
	}
	transpiler.Compile(flag.Arg(0))
}
