package checker

import (
	"go/ast"
	"go/parser"
	"go/token"
	gotypes "go/types"
	"os"
	"path/filepath"
	"testing"

	"github.com/lucasavila00/tgo/internal/checker/types"
	"golang.org/x/tools/go/packages"
)

func TestLoaderCachesPackageIdentity(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "main_tgo.go")
	content := []byte("package p\nfunc call(v int) (int,error) {return v,nil}\nfunc f() error {_=call(call(1)!)!!;return nil}\n")
	if err := os.WriteFile(filename, content, 0600); err != nil {
		t.Fatal(err)
	}
	metadata := &packages.Package{ID: "example.com/p", PkgPath: "example.com/p", GoFiles: []string{filename}, CompiledGoFiles: []string{filename}}
	loader := NewLoader(dir, []*packages.Package{metadata}, map[string]string{"main_tgo.go": "main.tgo"}, make(map[string]bool))
	first, err := loader.Check(metadata)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Operations) != 2 {
		t.Fatalf("operations: %d", len(first.Operations))
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

func TestCheckGoKeepsStockObjects(t *testing.T) {
	dir := t.TempDir()
	for name, source := range map[string]string{"go.mod": "module example.com/p\n\ngo 1.27.1\n", "main.go": "package p\nimport \"fmt\"\nvar Value = fmt.Sprint(1)\n"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	roots, err := packages.Load(&packages.Config{Dir: dir, Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports | packages.NeedDeps | packages.NeedExportFile | packages.NeedModule}, ".")
	if err != nil {
		t.Fatal(err)
	}
	if len(roots) != 1 || len(roots[0].Errors) != 0 {
		t.Fatalf("packages: %v", roots)
	}
	stock := roots[0]
	original := stock.Types.Scope().Lookup("Value")
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
	ast.Inspect(stock.Syntax[0], func(node ast.Node) bool {
		if id, ok := node.(*ast.Ident); ok && id.Name == "Value" {
			if stock.TypesInfo.Defs[id] != original || checked.TypesInfo.Defs[id] == nil {
				t.Fatal("binding information missing or changed")
			}
		}
		return true
	})
	if checked.Fset != stock.Fset || checked.Syntax[0] != stock.Syntax[0] {
		t.Fatal("ordinary syntax was replaced")
	}
}

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
			file, err := parser.ParseFile(fset, "main.go", "package p\n"+source, parser.SkipObjectResolution)
			if err != nil {
				t.Fatal(err)
			}
			stock := gotypes.Config{}
			_, stockErr := stock.Check("example.com/p", fset, []*ast.File{file}, nil)
			private := types.Config{}
			_, privateErr := private.Check("example.com/p", fset, []*ast.File{file}, nil)
			if (stockErr == nil) != (privateErr == nil) {
				t.Fatalf("stock error: %v; private error: %v", stockErr, privateErr)
			}
		})
	}
}
