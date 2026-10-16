package types_test

import (
	"go/token"
	"testing"

	"github.com/lucasavila00/tgo/internal/ast"
	"github.com/lucasavila00/tgo/internal/checker/types"
	"github.com/lucasavila00/tgo/internal/frontend/parser"
)

// TestInheritedConstantSyntax keeps each iota context on its own checked syntax.
func TestInheritedConstantSyntax(t *testing.T) {
	info, err := checkTGo(t, `type E enum { V }
const (
 A = len([iota+1]E{E.V{}})
 B
 C
)`)
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[ast.Expr]bool)
	for object, initializer := range info.ConstInitializers {
		want := int64(object.Name()[0]-'A') + 1
		call := initializer.Value.(*ast.CallExpr)
		array := info.TypeOf(call.Args[0]).(*types.Array)
		if array.Len() != want || info.Types[call].Value.String() != object.Val().String() {
			t.Fatalf("%s: array=%d, expression=%v, constant=%v",
				object.Name(), array.Len(), info.Types[call].Value, object.Val())
		}
		if seen[call] {
			t.Fatal("inherited declarations share checked expression nodes")
		}
		seen[call] = true
	}
	if len(seen) != 3 || len(info.EnumConstructors) != 3 {
		t.Fatalf("initializers=%d, constructors=%d", len(seen), len(info.EnumConstructors))
	}
}

// TestInheritedConstantTypes checks copied generic types without changing the source declaration spans.
func TestInheritedConstantTypes(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseTGoFile(fset, "input.tgo", "input.tgo", `package p
type E enum { V }
type Number[T any] int
const (
 A Number[[len([iota+1]E{E.V{}})]byte] = iota
 B
 C
)`, 0)
	if err != nil {
		t.Fatal(err)
	}
	declaration := file.Decls[2].(*ast.GenDecl)
	ends := make([]token.Pos, len(declaration.Specs))
	for index, specification := range declaration.Specs {
		ends[index] = specification.End()
	}
	info := &types.Info{Types: make(map[ast.Expr]types.TypeAndValue)}
	config := types.Config{}
	if _, err := config.Check("example.com/p", fset, []*ast.File{file}, info); err != nil {
		t.Fatal(err)
	}
	for object, initializer := range info.ConstInitializers {
		want := int64(object.Name()[0]-'A') + 1
		argument := object.Type().(*types.Named).TypeArgs().At(0).(*types.Array)
		if argument.Len() != want || !types.Identical(info.TypeOf(initializer.Type), object.Type()) {
			t.Fatalf("%s: wrong inherited type %s", object.Name(), object.Type())
		}
	}
	for index, specification := range declaration.Specs {
		if specification.End() != ends[index] {
			t.Fatal("checking changed a source declaration span")
		}
		value := specification.(*ast.ValueSpec)
		if index > 0 && (value.Type != nil || len(value.Values) != 0) {
			t.Fatal("checking filled an inherited source declaration")
		}
	}
}

// TestInheritedConstantBindings keeps separate local definitions in each copied function body.
func TestInheritedConstantBindings(t *testing.T) {
	info, err := checkTGo(t, `const (
 A = len([1]func()int{func()int { local := 1; return local }})
 B
 _
)`)
	if err != nil {
		t.Fatal(err)
	}
	locals := make(map[types.Object]bool)
	for identifier, object := range info.Defs {
		if identifier.Name == "local" {
			locals[object] = true
		}
	}
	if len(locals) != 3 || len(info.ConstInitializers) != 3 {
		t.Fatalf("local definitions=%d, constant initializers=%d", len(locals), len(info.ConstInitializers))
	}
}

// TestOrdinaryGoConstantsKeepSyntax preserves stock Go checking for files parsed as Go.
func TestOrdinaryGoConstantsKeepSyntax(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "input.go", "package p; const (A = iota; B)", 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{}
	config := types.Config{}
	if _, err := config.Check("example.com/p", fset, []*ast.File{file}, info); err != nil {
		t.Fatal(err)
	}
	if info.ConstInitializers != nil {
		t.Fatal("ordinary Go constants acquired TGo syntax records")
	}
}
