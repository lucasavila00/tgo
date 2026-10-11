package tests

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lucasavila00/tgo/pkg/transpiler"
)

func TestTypedTemporaryRejected(t *testing.T) {
	entries, err := os.ReadDir("testdata/typed-temporaries-rejected")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		t.Run(entry.Name(), func(t *testing.T) {
			dir := filepath.Join("testdata/typed-temporaries-rejected", entry.Name())
			expected, err := os.ReadFile(filepath.Join(dir, "diagnostic.txt"))
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				actual := recover()
				if fmt.Sprint(actual) != strings.TrimSpace(string(expected)) {
					t.Fatalf("diagnostic: %v, want %s", actual, expected)
				}
			}()
			transpiler.Compile(dir)
		})
	}
}

func TestTypedTemporaryReference(t *testing.T) {
	for _, name := range []string{"ContextualTypes", "VariadicGeneric", "TypeNameIdentity"} {
		t.Run(name, func(t *testing.T) { checkTypedTemporaryReference(t, name) })
	}
}

func checkTypedTemporaryReference(t *testing.T, name string) {
	t.Helper()
	fixture := filepath.Join("testdata/propagation", name)
	generated := t.TempDir()
	files := transpiler.Compile(fixture)
	file, err := parser.ParseFile(token.NewFileSet(), "main_tgo.go", files["main_tgo.go"], 0)
	if err != nil {
		t.Fatal(err)
	}
	ast.Inspect(file, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.AssignStmt:
			if n.Tok == token.DEFINE {
				for _, lhs := range n.Lhs {
					if name, ok := lhs.(*ast.Ident); ok && strings.HasPrefix(name.Name, "tgo") {
						t.Errorf("generated temporary uses :=: %s", name.Name)
					}
				}
			}
		case *ast.ValueSpec:
			for _, name := range n.Names {
				if strings.HasPrefix(name.Name, "tgo") && n.Type == nil {
					t.Errorf("generated temporary has no explicit type: %s", name.Name)
				}
			}
		}
		return true
	})
	writeFiles(t, generated, files)
	want, err := os.ReadFile(filepath.Join(fixture, "stdout.txt"))
	if err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{generated, filepath.Join("testdata/propagation-reference", strings.ToLower(name))} {
		command := exec.Command("go", "run", ".")
		command.Dir = dir
		actual, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("run %s: %v\n%s", dir, err, actual)
		}
		if !bytes.Equal(actual, want) {
			t.Fatalf("output: %q, want %q", actual, want)
		}
	}
}
