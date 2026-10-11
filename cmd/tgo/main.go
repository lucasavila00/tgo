package main

import (
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/lucasavila00/tgo/pkg/transpiler"
)

func main() {
	output := flag.String("o", "", "output directory")
	flag.Parse()
	if flag.NArg() != 1 {
		panic("usage: tgo [-o output-directory] source-directory")
	}
	files := transpiler.Compile(flag.Arg(0))
	inPlace := *output == ""
	if inPlace {
		command := exec.Command("go", "env", "GOMOD")
		command.Dir = flag.Arg(0)
		module, err := command.Output()
		if err != nil {
			panic(err)
		}
		*output = filepath.Dir(strings.TrimSpace(string(module)))
	}
	for name, content := range files {
		if inPlace {
			saved, err := os.ReadFile(filepath.Join(*output, name))
			if err == nil && string(saved) == string(content) {
				continue
			}
			if !strings.HasSuffix(name, ".go") {
				continue
			}
		}
		path := filepath.Join(*output, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			panic(err)
		}
		if err := os.WriteFile(path, content, 0644); err != nil {
			panic(err)
		}
	}
}
