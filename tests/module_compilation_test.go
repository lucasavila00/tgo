package tests

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/lucasavila00/tgo/pkg/transpiler"
)

func TestCompileFromSubdirectory(t *testing.T) {
	base := t.TempDir()
	dependency := filepath.Join(base, "dependency")
	writeFiles(t, dependency, map[string][]byte{
		"go.mod":   []byte("module example.com/dependency\n\ngo 1.27.1\n"),
		"value.go": []byte("package dependency\nfunc Value() (int, error) { return 7, nil }\n"),
	})
	root := filepath.Join(base, "source")
	input := map[string][]byte{
		"go.mod": []byte("module example.com/source\n\ngo 1.27.1\n\nrequire example.com/dependency " +
			"v0.0.0\nreplace example.com/dependency => ../dependency\n"),

		"cmd/app/main.tgo": []byte(`package main

import (
	"fmt"

	"example.com/dependency"
)

func run() error {
	fmt.Println(dependency.Value()!)
	return nil
}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}
`),
		"cmd/app/main_tgo.go":    []byte("invalid stale generated file"),
		"asset.txt":              []byte("keep this file\n"),
		".git/ignored.txt":       []byte("not module output"),
		".worktrees/ignored.txt": []byte("not module output"),
	}
	writeFiles(t, root, input)
	transpiler.Compile(filepath.Join(root, "cmd", "app"))
	for name, want := range input {
		if name == "cmd/app/main_tgo.go" {
			continue
		}
		actual, err := os.ReadFile(filepath.Join(root, name))
		if err != nil || !bytes.Equal(actual, want) {
			t.Fatalf("compilation changed source file %s: %v", name, err)
		}
	}
	command := exec.Command("go", "run", "./cmd/app")
	command.Dir = root
	actual, err := command.CombinedOutput()
	if err != nil || string(actual) != "7\n" {
		t.Fatalf("generated program: %v, output %q", err, actual)
	}
}
