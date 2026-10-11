package main

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/lucasavila00/tgo/pkg/transpiler"
)

func main() {
	output := flag.String("o", "", "output directory")
	flag.Parse()
	if flag.NArg() != 1 || *output == "" {
		panic("usage: tgo -o output-directory source-directory")
	}
	for name, content := range transpiler.Compile(flag.Arg(0)) {
		path := filepath.Join(*output, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			panic(err)
		}
		if err := os.WriteFile(path, content, 0644); err != nil {
			panic(err)
		}
	}
}
