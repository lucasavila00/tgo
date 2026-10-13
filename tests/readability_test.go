package tests

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lucasavila00/tgo/pkg/transpiler"
)

// TestReadablePropagation checks generated names, discarded results, import bindings, and required goto blocks.
func TestReadablePropagation(t *testing.T) {
	work := copyFixture(t, "testdata/propagation/Readability")
	transpiler.Compile(work)
	for _, name := range []string{"main_tgo.go", "alias_tgo.go"} {
		content, err := os.ReadFile(filepath.Join(work, name))
		if err != nil {
			t.Fatal(err)
		}
		file, err := parser.ParseFile(token.NewFileSet(), name, content, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok {
				continue
			}
			checkReadableFunction(t, function)
		}
	}
}

// checkDiscardedPropagation checks that discarded values need no storage and the error keeps its source type.
func checkDiscardedPropagation(t *testing.T, function *ast.FuncDecl) {
	t.Helper()
	if len(function.Body.List) != 4 {
		t.Fatalf("discarded propagation has %d statements, want 4", len(function.Body.List))
	}
	declaration, ok := function.Body.List[0].(*ast.DeclStmt)
	if !ok {
		t.Fatal("discarded propagation has an extra block")
	}
	value := declaration.Decl.(*ast.GenDecl).Specs[0].(*ast.ValueSpec)
	if !strings.HasPrefix(value.Names[0].Name, "error_tgo") {
		t.Fatalf("error name does not show its role: %s", value.Names[0].Name)
	}
	if pointer, ok := value.Type.(*ast.StarExpr); !ok || pointer.X.(*ast.Ident).Name != "failure" {
		t.Fatal("call error lost its pointer type before the nil check")
	}
	assignment := function.Body.List[1].(*ast.AssignStmt)
	if len(assignment.Lhs) != 3 || assignment.Lhs[0].(*ast.Ident).Name != "_" ||
		assignment.Lhs[1].(*ast.Ident).Name != "_" {
		t.Fatal("discarded success results have storage")
	}
}

// checkFormatBinding checks that each generated error wrapper uses the expected fmt import binding.
func checkFormatBinding(t *testing.T, function *ast.FuncDecl, prefix string) {
	t.Helper()
	found := false
	ast.Inspect(function.Body, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if ok && selector.Sel.Name == "Errorf" {
			found = true
			name := selector.X.(*ast.Ident).Name
			if (prefix == "fmt_tgo" && !strings.HasPrefix(name, prefix)) || (prefix != "fmt_tgo" && name != prefix) {
				t.Errorf("%s uses %s.Errorf, want %s", function.Name.Name, name, prefix)
			}
		}
		return true
	})
	if !found {
		t.Fatalf("%s has no generated error wrapper", function.Name.Name)
	}
}

// checkReadableFunction checks the generated-code requirements for one readability fixture function.
func checkReadableFunction(t *testing.T, function *ast.FuncDecl) {
	t.Helper()
	switch function.Name.Name {
	case "discarded", "blank":
		checkDiscardedPropagation(t, function)
		if function.Name.Name == "discarded" {
			checkFormatBinding(t, function, "fmt")
		}
	case "shadowed", "closure":
		checkFormatBinding(t, function, "fmt_tgo")
	case "aliased":
		checkFormatBinding(t, function, "format")
	case "jumped":
		if _, ok := function.Body.List[1].(*ast.BlockStmt); !ok {
			t.Fatal("goto crosses an unprotected generated declaration")
		}
	}
}
