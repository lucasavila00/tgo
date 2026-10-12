package comprehensions

import (
	"github.com/lucasavila00/tgo/internal/lowering/shared"
	"go/ast"
	"go/token"

	"github.com/lucasavila00/tgo/internal/checker/types"
	"github.com/lucasavila00/tgo/internal/frontend"
)

// Expression constructs a collection at its source evaluation point.
func Expression(r *shared.Context, call *ast.CallExpr, record *frontend.Comprehension) shared.Result {
	result := r.Name()
	typ := r.TypeOf(call)
	destination := &shared.CollectionDestination{Result: result, Type: typ}
	r.RegisterComprehension(record, destination)
	r.ComprehensionBindings(record.Body)
	empty := &ast.CompositeLit{Type: r.TemporaryType(typ, call)}
	declaration := r.ResultStorage(result, typ, call, empty)
	output := []ast.Stmt{declaration}
	key, _ := types.ComprehensionTypes(typ)
	if key == nil && r.BuiltinHidden("append") {
		r.PrepareSliceImport()
		count := r.Name()
		destination.Grow = count
		output = append(output, shared.Variable(count, r.TemporaryType(types.Typ[types.Int], call), shared.Integer(0)))
	}
	if exact := exactComprehension(r, call, record, destination); exact != nil {
		output = exact
	} else {
		output = append(output, r.Block(record.Body))
	}
	r.CompleteExpression(call)
	return r.ResultOf(output, []ast.Expr{result})
}

// Output evaluates output operands and inserts them into the collection.
func Output(r *shared.Context, call *ast.CallExpr, destination *shared.CollectionDestination) shared.Result {
	r.CompleteExpression(call)
	beforeEvaluation := r.Operands(call.Args)
	before, values := beforeEvaluation.Statements, beforeEvaluation.Expressions()
	keyType, element := types.ComprehensionTypes(destination.Type)
	if keyType == nil {
		value := r.Name()
		before = append(before, shared.Variable(value, r.TemporaryType(element, call.Args[0]), values[0]))
		if destination.Index != nil {
			before = append(before, &ast.AssignStmt{Lhs: []ast.Expr{&ast.IndexExpr{X: destination.Result,
				Index: destination.Index}}, Tok: token.ASSIGN, Rhs: []ast.Expr{value}})
			return r.ResultOf(before, nil)
		}
		if destination.Grow != nil {
			return r.ResultOf(append(before, growComprehension(r, destination, value)...), nil)
		}
		appendCall := &ast.CallExpr{Fun: ast.NewIdent("append"), Args: []ast.Expr{destination.Result, value}}
		before = append(before, &ast.AssignStmt{Lhs: []ast.Expr{destination.Result}, Tok: token.ASSIGN,
			Rhs: []ast.Expr{appendCall}})
	} else {
		key, value := r.Name(), r.Name()
		before = append(before, shared.Variable(key, r.TemporaryType(keyType, call.Args[0]), values[0]))
		before = append(before, shared.Variable(value, r.TemporaryType(element, call.Args[1]), values[1]))
		before = append(before, &ast.AssignStmt{Lhs: []ast.Expr{&ast.IndexExpr{X: destination.Result,
			Index: key}}, Tok: token.ASSIGN, Rhs: []ast.Expr{value}})
	}

	return r.ResultOf(before, nil)
}

// growComprehension appends through slices.Grow when source hides append.
func growComprehension(r *shared.Context, destination *shared.CollectionDestination, value ast.Expr) []ast.Stmt {
	slicePackage := r.SlicePackage()
	result, count := destination.Result, destination.Grow
	grown := &ast.CallExpr{Fun: &ast.SelectorExpr{X: slicePackage, Sel: ast.NewIdent("Grow")},
		Args: []ast.Expr{result, shared.Integer(1)}}
	extended := &ast.SliceExpr{X: result, High: &ast.BinaryExpr{X: count, Op: token.ADD, Y: shared.Integer(1)}}
	slot := &ast.IndexExpr{X: result, Index: count}
	return []ast.Stmt{
		&ast.AssignStmt{Lhs: []ast.Expr{result}, Tok: token.ASSIGN, Rhs: []ast.Expr{grown}},
		&ast.AssignStmt{Lhs: []ast.Expr{result}, Tok: token.ASSIGN, Rhs: []ast.Expr{extended}},
		&ast.AssignStmt{Lhs: []ast.Expr{slot}, Tok: token.ASSIGN, Rhs: []ast.Expr{value}},
		&ast.IncDecStmt{X: count, Tok: token.INC},
	}
}
