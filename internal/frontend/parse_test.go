package frontend

import (
	"go/ast"
	"go/token"
	"testing"
)

func TestParsePropagationPositions(t *testing.T) {
	fset := token.NewFileSet()
	parsed, err := ParseFile(fset, "main_tgo.go", "main.tgo", []byte("package p\nfunc f() error { _ = call()!; return nil }"), make(map[string]bool))
	if err != nil {
		t.Fatal(err)
	}
	ast.Inspect(parsed.File, func(node ast.Node) bool {
		if call, ok := node.(*ast.CallExpr); ok && fset.File(call.Pos()) == nil {
			t.Fatalf("call has no source file: %#v", call)
		}
		return true
	})
}
