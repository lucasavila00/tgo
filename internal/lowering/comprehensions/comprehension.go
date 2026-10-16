package comprehensions

import (
	"github.com/lucasavila00/tgo/internal/ast"
	"github.com/lucasavila00/tgo/internal/lowering/shared"
	"go/token"

	"github.com/lucasavila00/tgo/internal/checker/types"
)

// Expression constructs a collection at its source evaluation point.
func Expression(r *shared.Context, expression *ast.ComprehensionExpr) shared.Result {
	result := r.Name()
	typ := r.TypeOf(expression)
	destination := &shared.CollectionDestination{Result: result, Type: typ}
	r.ComprehensionBindings(expression)
	empty := &ast.CompositeLit{Type: r.TemporaryType(typ, expression)}
	declaration := r.ResultStorage(result, typ, expression, empty)
	output := []ast.Stmt{declaration}
	key, _ := types.ComprehensionTypes(typ)
	if key == nil && r.BuiltinHidden("append") {
		r.PrepareSliceImport()
		count := r.Name()
		destination.Grow = count
		output = append(output, shared.Variable(count, r.TemporaryType(types.Typ[types.Int], expression), shared.Integer(0)))
	}
	if exact := exactComprehension(r, expression, destination); exact != nil {
		output = exact
	} else {
		body := &ast.BlockStmt{Lbrace: expression.Lbrace, Rbrace: expression.Rbrace,
			List: clauses(r, expression, 0, destination)}
		output = append(output, body)
	}
	r.CompleteExpression(expression)
	return r.ResultOf(output, []ast.Expr{result})
}

// clauses builds each generator and filter around the next clause's output.
func clauses(r *shared.Context, expression *ast.ComprehensionExpr, index int,
	destination *shared.CollectionDestination) []ast.Stmt {
	if index == len(expression.Clauses) {
		return r.ComprehensionStatements(Output(r, outputOperands(expression), destination).Statements)
	}
	switch clause := expression.Clauses[index].(type) {
	case *ast.RangeClause:
		body := &ast.BlockStmt{Lbrace: clause.Pos(), Rbrace: expression.Rbrace,
			List: clauses(r, expression, index+1, destination)}
		return r.ComprehensionRange(clause, body)
	case *ast.FilterClause:
		body := &ast.BlockStmt{Lbrace: clause.Pos(), Rbrace: expression.Rbrace,
			List: clauses(r, expression, index+1, destination)}
		condition := r.ComprehensionOperand(clause.Cond, false)
		statement := &ast.IfStmt{If: clause.If, Cond: condition.Expressions()[0], Body: body}
		if len(condition.Statements) == 0 {
			return []ast.Stmt{statement}
		}
		return []ast.Stmt{&ast.BlockStmt{List: append(condition.Statements, statement)}}
	}
	panic("unknown comprehension clause")
}

// outputOperands reads map keys and values without changing their source nodes.
func outputOperands(expression *ast.ComprehensionExpr) []ast.Expr {
	var operands []ast.Expr
	for _, output := range expression.Output {
		if pair, ok := output.(*ast.KeyValueExpr); ok {
			operands = append(operands, pair.Key, pair.Value)
		} else {
			operands = append(operands, output)
		}
	}
	return operands
}

// Output evaluates output operands and inserts them into the collection.
func Output(r *shared.Context, operands []ast.Expr, destination *shared.CollectionDestination) shared.Result {
	beforeEvaluation := r.Operands(operands)
	before, values := beforeEvaluation.Statements, beforeEvaluation.Expressions()
	keyType, element := types.ComprehensionTypes(destination.Type)
	if keyType == nil {
		value := r.Name()
		before = append(before, shared.Variable(value, r.TemporaryType(element, operands[0]), values[0]))
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
		before = append(before, shared.Variable(key, r.TemporaryType(keyType, operands[0]), values[0]))
		before = append(before, shared.Variable(value, r.TemporaryType(element, operands[1]), values[1]))
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
