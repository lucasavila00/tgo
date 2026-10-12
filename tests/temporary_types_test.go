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
			transpiler.Compile(copyFixture(t, dir))
		})
	}
}

func TestTypedTemporaryDiagnosticStable(t *testing.T) {
	fixture := copyFixture(t, "testdata/typed-temporaries-rejected/Imports--returned_function")
	expected, err := os.ReadFile(filepath.Join(fixture, "diagnostic.txt"))
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 16; i++ {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			defer func() {
				if actual := recover(); fmt.Sprint(actual) != strings.TrimSpace(string(expected)) {
					t.Fatalf("diagnostic: %v, want %s", actual, expected)
				}
			}()
			transpiler.Compile(fixture)
		})
	}
}

func TestTypedTemporaryReference(t *testing.T) {
	for _, name := range []string{"ContextualTypes", "VariadicGeneric", "TypeNameIdentity", "ReturnTypeScope"} {
		t.Run(name, func(t *testing.T) { checkTypedTemporaryReference(t, name) })
	}
}

func TestTypeNameIdentityOutputStable(t *testing.T) {
	fixture := "testdata/propagation/TypeNameIdentity"
	want, err := os.ReadFile(filepath.Join(fixture, "main_tgo.go"))
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4; i++ {
		work := copyFixture(t, fixture)
		transpiler.Compile(work)
		actual, err := os.ReadFile(filepath.Join(work, "main_tgo.go"))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(actual, want) {
			t.Fatalf("compilation %d differs from the snapshot", i+1)
		}
	}
}

func checkTypedTemporaryReference(t *testing.T, name string) {
	t.Helper()
	fixture := filepath.Join("testdata/propagation", name)
	generated := copyFixture(t, fixture)
	transpiler.Compile(generated)
	file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(generated, "main_tgo.go"), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	checkTemporaryDeclarations(t, file)
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

func checkTemporaryDeclarations(t *testing.T, file *ast.File) {
	t.Helper()
	ast.Inspect(file, func(node ast.Node) bool {
		checkTemporaryDeclaration(t, node)
		return true
	})
}

func checkTemporaryDeclaration(t *testing.T, node ast.Node) {
	t.Helper()
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
}
