package driver

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompileWritesOnlyOwnedGeneratedFiles(t *testing.T) {
	root := t.TempDir()
	files := map[string][]byte{
		"go.mod":      []byte("module example.com/stages\n\ngo 1.27.1\n"),
		"main.tgo":    []byte("package p\nvar Value = if true {Keep} else {0}\n"),
		"ordinary.go": []byte("package p; var Keep=2\n"),
		"main_tgo.go": []byte("invalid snapshot"),
		"asset.bin":   bytes.Repeat([]byte{42}, 1<<20),
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(root, name), content, 0600); err != nil {
			t.Fatal(err)
		}
	}
	output := Compile(root)
	for _, name := range []string{"main.tgo", "ordinary.go", "asset.bin"} {
		got, err := os.ReadFile(filepath.Join(root, name))
		if err != nil || !bytes.Equal(got, files[name]) {
			t.Fatalf("changed input %s: %v", name, err)
		}
	}
	got, err := os.ReadFile(filepath.Join(root, "main_tgo.go"))
	if err != nil || !bytes.Equal(got, output["main_tgo.go"]) || bytes.Equal(got, files["main.tgo"]) {
		t.Fatalf("generated disk output: %v, %s", err, got)
	}
}

func TestCompileReplacesFailedStage(t *testing.T) {
	root := t.TempDir()
	input := []byte("package p\nvar Value = missing\n")
	for name, content := range map[string][]byte{"go.mod": []byte("module example.com/retry\n\ngo 1.27.1\n"), "main.tgo": input} {
		if err := os.WriteFile(filepath.Join(root, name), content, 0600); err != nil {
			t.Fatal(err)
		}
	}
	var failure any
	func() { defer func() { failure = recover() }(); Compile(root) }()
	if failure == nil || !strings.Contains(fmt.Sprint(failure), "undefined: missing") {
		t.Fatalf("source error: %v", failure)
	}
	got, err := os.ReadFile(filepath.Join(root, "main_tgo.go"))
	if err != nil || !bytes.Equal(got, input) {
		t.Fatalf("failed source stage: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.tgo"), []byte("package p\nvar Value = 3\n"), 0600); err != nil {
		t.Fatal(err)
	}
	output := Compile(root)
	got, err = os.ReadFile(filepath.Join(root, "main_tgo.go"))
	if err != nil || !bytes.Equal(got, output["main_tgo.go"]) {
		t.Fatalf("retry output: %v", err)
	}
}

func TestCompileReturnsModuleUpdates(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GOFLAGS", "-mod=mod")
	for name, content := range map[string]string{"go.mod": "module example.com/metadata\n\ngo 1.27.1\n\nrequire golang.org/x/mod v0.42.0\n", "main.tgo": "package p\nimport \"golang.org/x/mod/modfile\"\nvar Value = modfile.File{}\n"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	output := Compile(root)
	sums, err := os.ReadFile(filepath.Join(root, "go.sum"))
	if err != nil || len(sums) == 0 || !bytes.Equal(sums, output["go.sum"]) {
		t.Fatalf("returned module sums: %v", err)
	}
}
