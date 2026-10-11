package tests

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lucasavila00/tgo/pkg/transpiler"
)

func TestCompileOverlayKeepsEmbeddedAssets(t *testing.T) {
	root := t.TempDir()
	files := map[string][]byte{
		"go.mod":      []byte("module example.com/embedded\n\ngo 1.27.1\n"),
		"message.txt": []byte("hello from embed\n"),
		"main.tgo":    []byte("package main\nimport (\"embed\"; \"fmt\")\n//go:embed message.txt\nvar files embed.FS\nfunc run() error { fmt.Print(string(files.ReadFile(\"message.txt\")!)); return nil }\nfunc main(){if err:=run();err!=nil {panic(err)}}\n"),
		"main_tgo.go": []byte("invalid stale snapshot"),
	}
	writeFiles(t, root, files)
	output := transpiler.Compile(root)
	assertOriginalFiles(t, root, files)
	out := t.TempDir()
	writeFiles(t, out, output)
	command := exec.Command("go", "run", ".")
	command.Dir = out
	got, err := command.CombinedOutput()
	if err != nil || string(got) != "hello from embed\n" {
		t.Fatalf("embedded output: %v, %s", err, got)
	}
}

func TestCompileOverlayFailureKeepsSource(t *testing.T) {
	for _, source := range []string{"package p\nfunc f() int {return if true { 1 } else { missing }}\n", "package p\nvar Value int = \"wrong\"\n"} {
		t.Run(fmt.Sprint(len(source)), func(t *testing.T) {
			root := t.TempDir()
			files := map[string][]byte{"go.mod": []byte("module example.com/failed\n\ngo 1.27.1\n"), "main.tgo": []byte(source), "main_tgo.go": []byte("invalid stale snapshot"), "asset.bin": bytes.Repeat([]byte{42}, 1024)}
			writeFiles(t, root, files)
			var failure any
			func() { defer func() { failure = recover() }(); transpiler.Compile(root) }()
			if failure == nil || !strings.Contains(fmt.Sprint(failure), "TGo:") {
				t.Fatalf("source failure: %v", failure)
			}
			assertOriginalFiles(t, root, files)
		})
	}
}

func assertOriginalFiles(t *testing.T, root string, files map[string][]byte) {
	t.Helper()
	for name, want := range files {
		got, err := os.ReadFile(filepath.Join(root, name))
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("changed source %s: %v", name, err)
		}
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != len(files) {
		t.Fatalf("source directory has %d entries, want %d", len(entries), len(files))
	}
}

func TestCompileOverlayKeepsCgoHeaders(t *testing.T) {
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
		"go.mod":   []byte("module example.com/cgooverlay\n\ngo 1.27.1\n"),
		"value.h":  []byte("static int value(void) { return 7; }\n"),
		"value.go": []byte("package main\n/*\n#include \"value.h\"\n*/\nimport \"C\"\nfunc value() (int,error) { return int(C.value()),nil }\n"),
		"main.tgo": []byte("package main\nimport \"fmt\"\nfunc run() error {fmt.Println(value()!);return nil}\nfunc main(){if err:=run();err!=nil {panic(err)}}\n"),
	}
	writeFiles(t, root, files)
	output := transpiler.Compile(root)
	assertOriginalFiles(t, root, files)
	out := t.TempDir()
	writeFiles(t, out, output)
	command = exec.Command("go", "run", ".")
	command.Dir = out
	got, err := command.CombinedOutput()
	if err != nil || string(got) != "7\n" {
		t.Fatalf("cgo output: %v, %s", err, got)
	}
}
