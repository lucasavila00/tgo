package lowering

import (
	"go/ast"
	"go/token"

	"github.com/lucasavila00/tgo/internal/checker/types"
)

func (r *rewrite) unsafeOperand(call *ast.CallExpr) bool {
	var id *ast.Ident
	switch function := ast.Unparen(call.Fun).(type) {
	case *ast.Ident:
		id = function
	case *ast.SelectorExpr:
		id = function.Sel
	}
	builtin, ok := r.source.checked.TypesInfo.Uses[id].(*types.Builtin)
	if !ok || builtin.Pkg() == nil || builtin.Pkg().Path() != "unsafe" {
		return false
	}
	switch builtin.Name() {
	case "Sizeof", "Alignof", "Offsetof":
		return true
	}
	return false
}

// unevaluatedCall retains source bindings and jump targets without running the
// operand. The builtin receives an expression with the same checked type.
func (r *rewrite) unevaluatedCall(call *ast.CallExpr) ([]ast.Stmt, ast.Expr) {
	operand := call.Args[0]
	base := operand
	selector, field := ast.Unparen(operand).(*ast.SelectorExpr)
	var builtinName string
	switch function := ast.Unparen(call.Fun).(type) {
	case *ast.Ident:
		builtinName = function.Name
	case *ast.SelectorExpr:
		builtinName = function.Sel.Name
	}
	if builtinName == "Offsetof" && field {
		base = selector.X
	}
	before, values := r.expression(operand)
	before = append(before, &ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent("_")}, Tok: token.ASSIGN, Rhs: values})
	guard := &ast.IfStmt{Cond: boolean(false), Body: &ast.BlockStmt{List: before}}
	pointer := &ast.StarExpr{X: r.temporaryType(r.typeOf(base), base)}
	zero := &ast.FuncLit{
		Type: &ast.FuncType{Params: &ast.FieldList{}, Results: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{r.name()}, Type: pointer}}}},
		Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{}}},
	}
	var replacement ast.Expr = &ast.StarExpr{X: &ast.CallExpr{Fun: zero}}
	if builtinName == "Offsetof" && field {
		copy := *selector
		copy.X = replacement
		replacement = &copy
	}
	copy := *call
	copy.Args = []ast.Expr{replacement}
	return []ast.Stmt{guard}, &copy
}
