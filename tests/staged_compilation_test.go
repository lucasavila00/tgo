package tests

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lucasavila00/tgo/pkg/transpiler"
)

// TestCompileStagesUseEmbeddedAssets checks that generated Go can read an embedded asset from the source module.
func TestCompileStagesUseEmbeddedAssets(t *testing.T) {
	root := t.TempDir()
	files := map[string][]byte{
		"go.mod":      []byte("module example.com/embedded\n\ngo 1.27.1\n"),
		"message.txt": []byte("hello from embed\n"),
		"main.tgo": []byte("package main\nimport (\"embed\"; \"fmt\")\n//go:embed message.txt\nvar files " +
			"embed.FS\nfunc run() error { " +
			"fmt.Print(string(files.ReadFile(\"message.txt\")!)); return nil }\nfunc " +
			"main(){if err:=run();err!=nil {panic(err)}}\n"),

		"main_tgo.go": []byte("invalid stale snapshot"),
	}
	writeFiles(t, root, files)
	transpiler.Compile(root)
	assertInputFiles(t, root, files)
	command := exec.Command("go", "run", ".")
	command.Dir = root
	got, err := command.CombinedOutput()
	if err != nil || string(got) != "hello from embed\n" {
		t.Fatalf("embedded output: %v, %s", err, got)
	}
}

// assertInputFiles checks that compilation preserves every supplied input except the generated output.
func assertInputFiles(t *testing.T, root string, files map[string][]byte) {
	t.Helper()
	for name, want := range files {
		if name == "main_tgo.go" {
			continue
		}
		got, err := os.ReadFile(filepath.Join(root, name))
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("changed source %s: %v", name, err)
		}
	}
}

// TestCompileStagesUseCgoHeaders checks that generated Go can use a cgo dependency with a local header.
func TestCompileStagesUseCgoHeaders(t *testing.T) {
	command := exec.Command("go", "env", "CGO_ENABLED")
	enabled, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(enabled)) != "1" {
		t.Skip("cgo is disabled")
	}
	root := t.TempDir()
	files := map[string][]byte{
		"go.mod":  []byte("module example.com/cgostages\n\ngo 1.27.1\n"),
		"value.h": []byte("static int value(void) { return 7; }\n"),
		"value.go": []byte("package main\n/*\n#include \"value.h\"\n*/\nimport \"C\"\nfunc value() (int,error) " +
			"{ return int(C.value()),nil }\n"),

		"main.tgo": []byte("package main\nimport \"fmt\"\nfunc run() error {fmt.Println(value()!);return " +
			"nil}\nfunc main(){if err:=run();err!=nil {panic(err)}}\n"),
	}
	writeFiles(t, root, files)
	transpiler.Compile(root)
	assertInputFiles(t, root, files)
	command = exec.Command("go", "run", ".")
	command.Dir = root
	got, err := command.CombinedOutput()
	if err != nil || string(got) != "7\n" {
		t.Fatalf("cgo output: %v, %s", err, got)
	}
}

// TestCompileStagesUseVendorSelection checks that compilation uses vendored Go without changing dependency files.
func TestCompileStagesUseVendorSelection(t *testing.T) {
	root := t.TempDir()
	files := map[string][]byte{
		"go.mod": []byte("module example.com/vendorstages\n\ngo 1.27.1\n\nrequire example.com/value " +
			"v1.0.0\n"),

		"vendor/modules.txt": []byte("# example.com/value v1.0.0\n## explicit; go 1.27.1\nexample.com/value\n"),

		"vendor/example.com/value/value.tgo":    []byte("invalid TGo inside a dependency"),
		"vendor/example.com/value/value_tgo.go": []byte("package value\nfunc Value()(int,error){return 9,nil}\n"),
		"main.tgo": []byte("package main\nimport (\"fmt\";\"example.com/value\")\nfunc " +
			"run()error{fmt.Println(value.Value()!);return nil}\nfunc main(){if " +
			"err:=run();err!=nil{panic(err)}}\n"),
	}
	writeFiles(t, root, files)
	transpiler.Compile(root)
	for name, want := range files {
		if name == "main_tgo.go" {
			continue
		}
		got, err := os.ReadFile(filepath.Join(root, name))
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("changed source %s: %v", name, err)
		}
	}
	command := exec.Command("go", "run", ".")
	command.Dir = root
	got, err := command.CombinedOutput()
	if err != nil || string(got) != "9\n" {
		t.Fatalf("vendor output: %v, %s", err, got)
	}
}

// TestCompileStagesUseWorkspaceDependency checks that compilation and generated Go use a workspace dependency.
func TestCompileStagesUseWorkspaceDependency(t *testing.T) {
	base := t.TempDir()
	writeFiles(t, base,
		map[string][]byte{"go.work": []byte("go 1.27.1\n\nuse (\n ./source\n ./dependency\n)\n"),
			"dependency/go.mod":   []byte("module example.com/value\n\ngo 1.27.1\n"),
			"dependency/value.go": []byte("package value\nfunc Value()(int,error){return 11,nil}\n")})
	root := filepath.Join(base, "source")
	files := map[string][]byte{
		"go.mod": []byte("module example.com/workspace\n\ngo 1.27.1\n\nrequire example.com/value v0.0.0\n"),
		"main.tgo": []byte("package main\nimport (\"fmt\";\"example.com/value\")\nfunc " +
			"run()error{fmt.Println(value.Value()!);return nil}\nfunc main(){if " +
			"err:=run();err!=nil{panic(err)}}\n")}
	writeFiles(t, root, files)
	transpiler.Compile(root)
	generated, err := os.ReadFile(filepath.Join(root, "main_tgo.go"))
	if err != nil || len(generated) == 0 {
		t.Fatal("missing module output")
	}
	command := exec.Command("go", "run", ".")
	command.Dir = root
	got, err := command.CombinedOutput()
	if err != nil || string(got) != "11\n" {
		t.Fatalf("workspace output: %v, %s", err, got)
	}
}
