package lowering

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPackageFunctionInsertionKeepsSource(t *testing.T) {
	dir := t.TempDir()
	input := `package example
func next() int { return 7 }
var functions = []func() int{func() int { return next() }}
`
	for name, content := range map[string]string{
		"go.mod":  "module example.com/functions\n\ngo 1.27.1\n",
		"main.go": input,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	source, err := Load(dir, ".")
	if err != nil {
		t.Fatal(err)
	}
	file := source.Package.Syntax[0]
	var before bytes.Buffer
	if err := format.Node(&before, source.Package.Fset, file); err != nil {
		t.Fatal(err)
	}
	var target Site
	for site := range source.expressions {
		if input[site.Start:site.End] == "next()" {
			target = site
		}
	}
	if target.Kind == "" {
		t.Fatal("missing call")
	}
	callback := func([]ast.Expr) []ast.Stmt {
		return []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: "9"}}}}
	}
	first, err := source.InsertAfter(target, callback)
	if err != nil {
		t.Fatal(err)
	}
	second, err := source.InsertAfter(target, callback)
	if err != nil {
		t.Fatal(err)
	}
	name := filepath.Join(dir, "main.go")
	if !bytes.Equal(first[name], second[name]) || !strings.Contains(string(first[name]), "return 9") {
		t.Fatalf("insertion changed across calls:\n%s\n%s", first[name], second[name])
	}
	var after bytes.Buffer
	if err := format.Node(&after, source.Package.Fset, file); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before.Bytes(), after.Bytes()) {
		t.Fatalf("source AST changed:\n%s", after.String())
	}
}

func TestPropagationEmissionKeepsSource(t *testing.T) {
	dir := t.TempDir()
	input := `package example
func next() error { return nil }
func marker(err error) {}
var functions = []func() error{func() error { marker(next()); return nil }}
`
	for name, content := range map[string]string{
		"go.mod":  "module example.com/functions\n\ngo 1.27.1\n",
		"main.go": input,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	source, err := Load(dir, ".")
	if err != nil {
		t.Fatal(err)
	}
	var before bytes.Buffer
	if err := format.Node(&before, source.Package.Fset, source.Package.Syntax[0]); err != nil {
		t.Fatal(err)
	}
	operations := map[string]Propagation{"marker": {CallName: "next"}}
	first, err := source.LowerPropagations(operations)
	if err != nil {
		t.Fatal(err)
	}
	second, err := source.LowerPropagations(operations)
	if err != nil {
		t.Fatal(err)
	}
	name := filepath.Join(dir, "main.go")
	if !bytes.Equal(first[name], second[name]) || strings.Contains(string(first[name]), "marker(") {
		t.Fatalf("propagation changed across calls:\n%s\n%s", first[name], second[name])
	}
	var after bytes.Buffer
	if err := format.Node(&after, source.Package.Fset, source.Package.Syntax[0]); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before.Bytes(), after.Bytes()) {
		t.Fatalf("source AST changed:\n%s", after.String())
	}
}
