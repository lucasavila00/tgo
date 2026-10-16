package lowering

import (
	"bytes"
	"github.com/lucasavila00/tgo/internal/ast"
	"github.com/lucasavila00/tgo/internal/format"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lucasavila00/tgo/internal/checker"
	"golang.org/x/tools/go/packages"
)

// TestPackageFunctionInsertionKeepsSource checks insertion in package-initializer closures preserves the checked AST.
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
	file := source.inventory.Checked.Syntax[0]
	var before bytes.Buffer
	if err := format.Node(&before, source.inventory.Checked.Fset, file); err != nil {
		t.Fatal(err)
	}
	var target Site
	for site := range source.inventory.Expressions {
		if input[site.Start:site.End] == "next()" {
			target = Site(site)
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
	if err := format.Node(&after, source.inventory.Checked.Fset, file); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before.Bytes(), after.Bytes()) {
		t.Fatalf("source AST changed:\n%s", after.String())
	}
}

// TestPropagationEmissionKeepsSource checks propagation in package-initializer closures preserves the checked AST.
func TestPropagationEmissionKeepsSource(t *testing.T) {
	dir := t.TempDir()
	input := `package example
func next() error { return nil }
var functions = []func() error{func() error { next()!; return nil }}
`
	for name, content := range map[string]string{
		"go.mod":      "module example.com/functions\n\ngo 1.27.1\n",
		"main_tgo.go": input,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	filename := filepath.Join(dir, "main_tgo.go")
	metadata := &packages.Package{ID: "example.com/functions", PkgPath: "example.com/functions",
		GoFiles: []string{filename}}
	sizes, err := checker.TargetSizes(dir)
	if err != nil {
		t.Fatal(err)
	}
	loader := checker.NewLoader(dir, []*packages.Package{metadata},
		map[string]string{"main_tgo.go": "main.tgo"}, sizes)
	checked, err := loader.Check(metadata)
	if err != nil {
		t.Fatal(err)
	}
	source, err := SourceFromCheckedPackage(checked, dir)
	if err != nil {
		t.Fatal(err)
	}
	var before bytes.Buffer
	if err := ast.Fprint(&before, source.inventory.Checked.Fset, source.inventory.Checked.Syntax[0], nil); err != nil {
		t.Fatal(err)
	}
	first, err := source.LowerPropagations()
	if err != nil {
		t.Fatal(err)
	}
	second, err := source.LowerPropagations()
	if err != nil {
		t.Fatal(err)
	}
	name := filepath.Join(dir, "main_tgo.go")
	if !bytes.Equal(first[name], second[name]) || strings.Contains(string(first[name]), "tgoPropagation") {
		t.Fatalf("propagation changed across calls:\n%s\n%s", first[name], second[name])
	}
	var after bytes.Buffer
	if err := ast.Fprint(&after, source.inventory.Checked.Fset, source.inventory.Checked.Syntax[0], nil); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before.Bytes(), after.Bytes()) {
		t.Fatalf("source AST changed:\n%s", after.String())
	}
}
