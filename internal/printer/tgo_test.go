package printer

import (
	"bytes"
	stock "go/format"
	"go/token"
	"testing"

	"github.com/lucasavila00/tgo/internal/ast"
)

// TestPrivateGoOutput checks that ordinary output uses Go's printer rules with private node identities.
func TestPrivateGoOutput(t *testing.T) {
	file := &ast.File{Name: ast.NewIdent("p"), Decls: []ast.Decl{
		&ast.GenDecl{Tok: token.VAR, Specs: []ast.Spec{
			&ast.ValueSpec{Names: []*ast.Ident{ast.NewIdent("values")}, Values: []ast.Expr{
				&ast.CompositeLit{Type: &ast.ArrayType{Elt: ast.NewIdent("int")},
					Elts: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: "1"},
						&ast.BasicLit{Kind: token.INT, Value: "2"}}},
			}},
		}},
	}}
	var output bytes.Buffer
	if err := (&Config{Mode: UseSpaces | TabIndent, Tabwidth: 8}).Fprint(&output, token.NewFileSet(), file); err != nil {
		t.Fatal(err)
	}
	want, err := stock.Source([]byte("package p\nvar values=[]int{1,2}"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(output.Bytes(), want) {
		t.Fatalf("output:\n%s\nwant:\n%s", output.Bytes(), want)
	}
}
