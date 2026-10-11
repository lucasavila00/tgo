package tests

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/lucasavila00/tgo/pkg/transpiler"
	"golang.org/x/mod/modfile"
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
		"go.mod":                 []byte("module example.com/source\n\ngo 1.27.1\n\nrequire example.com/dependency v0.0.0\nreplace example.com/dependency => ../dependency\n"),
		"cmd/app/main.tgo":       []byte("package main\nimport (\"fmt\"; \"example.com/dependency\")\nfunc run() error { fmt.Println(dependency.Value()!); return nil }\nfunc main() { if err := run(); err != nil { panic(err) } }\n"),
		"cmd/app/main_tgo.go":    []byte("invalid stale generated file"),
		"asset.txt":              []byte("keep this file\n"),
		".git/ignored.txt":       []byte("not module output"),
		".worktrees/ignored.txt": []byte("not module output"),
	}
	writeFiles(t, root, input)
	output := transpiler.Compile(filepath.Join(root, "cmd", "app"))
	for _, name := range []string{"asset.txt", "cmd/app/main.tgo"} {
		if !bytes.Equal(output[name], input[name]) {
			t.Fatalf("output changed %s", name)
		}
	}
	for _, name := range []string{".git/ignored.txt", ".worktrees/ignored.txt"} {
		if _, exists := output[name]; exists {
			t.Fatalf("output includes %s", name)
		}
	}
	module, err := modfile.Parse("go.mod", output["go.mod"], nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(module.Replace) != 1 || module.Replace[0].New.Path != dependency {
		t.Fatalf("replacement does not refer to source dependency: %s", output["go.mod"])
	}
	for name, want := range input {
		actual, err := os.ReadFile(filepath.Join(root, name))
		if err != nil || !bytes.Equal(actual, want) {
			t.Fatalf("compilation changed source file %s: %v", name, err)
		}
	}
	dir := t.TempDir()
	writeFiles(t, dir, output)
	command := exec.Command("go", "run", "./cmd/app")
	command.Dir = dir
	actual, err := command.CombinedOutput()
	if err != nil || string(actual) != "7\n" {
		t.Fatalf("generated program: %v, output %q", err, actual)
	}
}
