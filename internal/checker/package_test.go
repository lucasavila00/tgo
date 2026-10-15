package checker

import (
	"github.com/lucasavila00/tgo/internal/ast"
	"github.com/lucasavila00/tgo/internal/checker/types"
	"github.com/lucasavila00/tgo/internal/frontend/parser"
	stockast "go/ast"
	stockparser "go/parser"
	"go/token"
	gotypes "go/types"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/packages"
)

// TestCheckConditionalModule checks an if value in a module with an imported dependency.
func TestCheckConditionalModule(t *testing.T) {
	dir := t.TempDir()
	for name, source := range map[string]string{
		"go.mod": "module example.com/conditional\n\ngo 1.27.1\n",
		"main_tgo.go": `package main
import "fmt"
func choose(ok bool) string { return if ok { "yes" } else { "no" } }
func main() { fmt.Println(choose(true)) }
`,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	roots, err := packages.Load(&packages.Config{Dir: dir, Mode: packages.NeedName | packages.NeedFiles |
		packages.NeedCompiledGoFiles | packages.NeedImports | packages.NeedDeps | packages.NeedExportFile |
		packages.NeedModule}, "./...")
	if err != nil {
		t.Fatal(err)
	}
	if len(roots) != 1 {
		t.Fatalf("packages: %d", len(roots))
	}
	sizes, err := TargetSizes(dir)
	if err != nil {
		t.Fatal(err)
	}
	loader := NewLoader(dir, roots, map[string]string{"main_tgo.go": "main.tgo"}, sizes)
	pkg, err := loader.Check(roots[0])
	if err != nil {
		t.Fatal(err)
	}
	conditionals := 0
	for _, file := range pkg.Syntax {
		ast.Inspect(file, func(node ast.Node) bool {
			if _, ok := node.(*ast.IfExpr); ok {
				conditionals++
			}
			return true
		})
	}
	if conditionals != 1 {
		t.Fatalf("conditionals: %d", conditionals)
	}
}

// TestLoaderCachesPackageIdentity checks distinct package and test identities and reuse after source removal.
func TestLoaderCachesPackageIdentity(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "main_tgo.go")
	content := []byte("package p\nfunc call(v int) (int,error) {return v,nil}\nfunc f() error " +
		"{_=call(call(1)!)!!;return nil}\n")
	if err := os.WriteFile(filename, content, 0600); err != nil {
		t.Fatal(err)
	}
	metadata := &packages.Package{ID: "example.com/p", PkgPath: "example.com/p",
		GoFiles: []string{filename}, CompiledGoFiles: []string{filename}}
	sizes, err := TargetSizes(dir)
	if err != nil {
		t.Fatal(err)
	}
	loader := NewLoader(dir, []*packages.Package{metadata}, map[string]string{"main_tgo.go": "main.tgo"},
		sizes)
	first, err := loader.Check(metadata)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.CallNames) != 2 {
		t.Fatalf("source call names: %d", len(first.CallNames))
	}
	variant := *metadata
	variant.ID = "example.com/p [example.com/p.test]"
	other, err := loader.Check(&variant)
	if err != nil {
		t.Fatal(err)
	}
	if first.Types == other.Types {
		t.Fatal("test variant shares source package identity")
	}
	if err := os.Remove(filename); err != nil {
		t.Fatal(err)
	}
	cached, err := loader.Check(metadata)
	if err != nil || cached != first {
		t.Fatalf("cached package: %p, error %v", cached, err)
	}
	cached, err = loader.Check(&variant)
	if err != nil || cached != other {
		t.Fatalf("cached test variant: %p, error %v", cached, err)
	}
	if len(loader.packages) != 2 {
		t.Fatalf("checked packages: %d", len(loader.packages))
	}
}

// TestCheckGoKeepsStockObjects checks that private checking retains syntax and leaves stock Go bindings unchanged.
func TestCheckGoKeepsStockObjects(t *testing.T) {
	dir := t.TempDir()
	for name, source := range map[string]string{"go.mod": "module example.com/p\n\ngo 1.27.1\n",
		"main.go": "package p\nimport \"fmt\"\nvar Value = fmt.Sprint(1)\n"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	roots, err := packages.Load(&packages.Config{Dir: dir, Mode: packages.NeedName | packages.NeedFiles |
		packages.NeedCompiledGoFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo |
		packages.NeedImports | packages.NeedDeps | packages.NeedExportFile | packages.NeedModule}, ".")
	if err != nil {
		t.Fatal(err)
	}
	if len(roots) != 1 || len(roots[0].Errors) != 0 {
		t.Fatalf("packages: %v", roots)
	}
	stock := roots[0]
	original := stock.Types.Scope().Lookup("Value")
	originalSyntax, originalInfo := stock.Syntax[0], stock.TypesInfo
	checked, err := CheckGo(stock, dir)
	if err != nil {
		t.Fatal(err)
	}
	if stock.Types.Scope().Lookup("Value") != original {
		t.Fatal("stock binding changed")
	}
	if _, ok := original.Type().(*gotypes.Basic); !ok {
		t.Fatalf("stock type: %T", original.Type())
	}
	if _, ok := checked.Types.Scope().Lookup("Value").Type().(*types.Basic); !ok {
		t.Fatal("private object uses stock types")
	}
	stockast.Inspect(stock.Syntax[0], func(node stockast.Node) bool {
		if id, ok := node.(*stockast.Ident); ok && id.Name == "Value" && stock.TypesInfo.Defs[id] != original {
			t.Fatal("stock binding information changed")
		}
		return true
	})
	privateDefinition := checked.Types.Scope().Lookup("Value")
	ast.Inspect(checked.Syntax[0], func(node ast.Node) bool {
		if id, ok := node.(*ast.Ident); ok && id.Name == "Value" && checked.TypesInfo.Defs[id] != privateDefinition {
			t.Fatal("private binding is not attached to the private source node")
		}
		return true
	})
	if stock.Syntax[0] != originalSyntax || stock.TypesInfo != originalInfo || checked.Fset != stock.Fset {
		t.Fatal("stock syntax, type information, or file-set identity changed")
	}
}

// TestOrdinaryGoCheckingMatchesStock compares accepted and rejected ordinary Go with the stock checker.
func TestOrdinaryGoCheckingMatchesStock(t *testing.T) {
	for _, source := range []string{
		`func identity[T any](v T) T {return v}; var v=identity(3)`,
		`type Set[T comparable] = map[T]bool; var v=Set[int]{1:true}`,
		`func f() int {for i:=range 3 {if i>0 {return i}};return 0}`,
		`func f(v any) int {switch v:=v.(type) {case int:return v;default:return 0}}`,
		`type S struct{};func (S) F() int{return 1};var v=(S{}).F()`,
		`var v int = "wrong"`,
		`func f() int {}`,
		`func f() {var v=1}`,
		`func f() {goto end;v:=1;end:_=v}`,
	} {
		t.Run(source, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := stockparser.ParseFile(fset, "main.go", "package p\n"+source, stockparser.SkipObjectResolution)
			if err != nil {
				t.Fatal(err)
			}
			stock := gotypes.Config{}
			_, stockErr := stock.Check("example.com/p", fset, []*stockast.File{file}, nil)
			privateSet := token.NewFileSet()
			privateFile, err := parser.ParseFile(privateSet, "main.go", "package p\n"+source, parser.SkipObjectResolution)
			if err != nil {
				t.Fatal(err)
			}
			private := types.Config{}
			_, privateErr := private.Check("example.com/p", privateSet, []*ast.File{privateFile}, nil)
			if (stockErr == nil) != (privateErr == nil) {
				t.Fatalf("stock error: %v; private error: %v", stockErr, privateErr)
			}
		})
	}
}

// TestTargetSizesUseGoConfiguration checks that array sizes use the architecture selected by Go configuration.
func TestTargetSizesUseGoConfiguration(t *testing.T) {
	dir := t.TempDir()
	environment := filepath.Join(dir, "goenv")
	if err := os.WriteFile(environment, []byte("GOARCH=386\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOENV", environment)
	t.Setenv("GOARCH", "")
	filename := filepath.Join(dir, "main_tgo.go")
	source := `package p
import "unsafe"
var storage [4-unsafe.Sizeof(uintptr(0))]byte
func next() error {return nil}
func run() error {next()!;return nil}
`
	if err := os.WriteFile(filename, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	metadata := &packages.Package{ID: "example.com/p", PkgPath: "example.com/p", GoFiles: []string{filename}}
	sizes, err := TargetSizes(dir)
	if err != nil {
		t.Fatal(err)
	}
	loader := NewLoader(dir, []*packages.Package{metadata}, map[string]string{"main_tgo.go": "main.tgo"},
		sizes)
	checked, err := loader.Check(metadata)
	if err != nil {
		t.Fatal(err)
	}
	if length := checked.Types.Scope().Lookup("storage").Type().(*types.Array).Len(); length != 0 {
		t.Fatalf("array length: %d; want 0", length)
	}
}
