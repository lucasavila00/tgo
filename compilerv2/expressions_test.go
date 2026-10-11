package compilerv2

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestExpressionCallOrder(t *testing.T) {
	dir := t.TempDir()
	input := `package main
import "fmt"
func left() int { fmt.Print("L"); return 1 }
func right() int { fmt.Print("R"); return 2 }
func use(a, b int) { fmt.Print("U") }
func main() { use(left(), right()) }
`
	for name, content := range map[string]string{"go.mod": "module example.com/order\n\ngo 1.27.1\n", "main.go": input} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	source, err := Load(dir, ".")
	if err != nil {
		t.Fatal(err)
	}
	var target ast.Expr
	for site, expr := range source.expressions {
		if input[site.Start:site.End] == "right()" {
			target = expr
		}
	}
	if target == nil {
		t.Fatal("missing expression")
	}
	r := &rewrite{source: source, target: target, names: map[string]bool{}, after: func([]ast.Expr) []ast.Stmt {
		return []ast.Stmt{&ast.ExprStmt{X: &ast.CallExpr{Fun: &ast.SelectorExpr{X: ast.NewIdent("fmt"), Sel: ast.NewIdent("Print")}, Args: []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: `"M"`}}}}}
	}}
	file := *source.Package.Syntax[0]
	file.Decls = append([]ast.Decl(nil), file.Decls...)
	for i, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "main" {
			copy := *fn
			before, refs := r.expression(fn.Body.List[0].(*ast.ExprStmt).X)
			copy.Body = &ast.BlockStmt{List: append(before, &ast.ExprStmt{X: refs[0]})}
			file.Decls[i] = &copy
		}
	}
	var output bytes.Buffer
	if err := format.Node(&output, source.Package.Fset, &file); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), output.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", ".")
	cmd.Dir = dir
	actual, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compile: %v\n%s\n%s", err, actual, output.String())
	}
	if string(actual) != "LRMU" {
		t.Fatalf("got %q, want LRMU\n%s", actual, output.String())
	}
}
