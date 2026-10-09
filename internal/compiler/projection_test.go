package compiler

import (
	"go/ast"
	"go/token"
	"go/types"
	"testing"
)

func TestProjectionMarksGeneratedFunctionFacts(t *testing.T) {
	t.Parallel()
	generatedExpression := &ast.BasicLit{Kind: token.INT, Value: "1"}
	generatedFunction := &ast.FuncDecl{
		Name: ast.NewIdent("generated"),
		Type: &ast.FuncType{Params: new(ast.FieldList)},
		Body: &ast.BlockStmt{List: []ast.Stmt{
			&ast.ExprStmt{X: generatedExpression},
		}},
	}
	sourceExpression := &ast.BasicLit{Kind: token.INT, Value: "2"}
	generatedType := ast.NewIdent("int")
	generatedDeclaration := &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{&ast.TypeSpec{
			Name: ast.NewIdent("Payload"), Type: generatedType,
		}},
	}
	info := newInfo()
	info.Types[generatedExpression] = types.TypeAndValue{Type: types.Typ[types.Int]}
	info.Types[sourceExpression] = types.TypeAndValue{Type: types.Typ[types.String]}
	info.Types[generatedType] = types.TypeAndValue{Type: types.Typ[types.Bool]}
	unit := &packageUnit{
		info: info,
		generated: map[ast.Decl]bool{
			generatedFunction:    true,
			generatedDeclaration: true,
		},
	}

	generated := make(map[types.Type]bool)
	newProjectionFacts(unit).RangeTypes(func(
		_ token.Pos,
		_ token.Pos,
		value types.TypeAndValue,
		synthetic bool,
	) {
		generated[value.Type] = synthetic
	})
	if !generated[types.Typ[types.Int]] {
		t.Fatal("generated function expression is not marked")
	}
	if generated[types.Typ[types.String]] {
		t.Fatal("source expression is marked as generated")
	}
	if generated[types.Typ[types.Bool]] {
		t.Fatal("generated payload type is marked as a function fact")
	}
}
