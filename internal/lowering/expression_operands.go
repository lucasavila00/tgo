package lowering

import (
	"github.com/lucasavila00/tgo/internal/checker/types"
	"go/ast"
	"go/token"
)

func (r *rewrite) binaryExpression(e *ast.BinaryExpr) ([]ast.Stmt, ast.Expr) {
	if e.Op == token.LAND || e.Op == token.LOR {
		return r.logicalExpression(e)
	}
	statementsEvaluation := r.operands([]ast.Expr{e.X, e.Y})
	statements, values := statementsEvaluation.statements, statementsEvaluation.expressions()
	output := *e
	output.X = values[0]
	output.Y = values[1]
	return statements, &output
}

func (r *rewrite) logicalExpression(expression *ast.BinaryExpr) ([]ast.Stmt, ast.Expr) {
	leftEvaluation := r.expression(expression.X)
	leftStatements, leftValues := leftEvaluation.statements, leftEvaluation.expressions()
	rightEvaluation := r.expression(expression.Y)
	rightStatements, rightValues := rightEvaluation.statements, rightEvaluation.expressions()

	output := *expression
	output.X = leftValues[0]
	output.Y = rightValues[0]
	if len(rightStatements) == 0 {
		return leftStatements, &output
	}

	result := r.name()
	statements := append(leftStatements, variable(result, r.temporaryType(r.typeOf(expression), expression), nil))
	statements = append(statements, &ast.AssignStmt{Lhs: []ast.Expr{result}, Tok: token.ASSIGN, Rhs: leftValues})
	condition := ast.Expr(result)
	if expression.Op == token.LOR {
		condition = &ast.UnaryExpr{Op: token.NOT, X: result}
	}
	rightStatements = append(rightStatements, &ast.AssignStmt{Lhs: []ast.Expr{result}, Tok: token.ASSIGN,
		Rhs: rightValues})
	statements = append(statements, &ast.IfStmt{Cond: condition, Body: &ast.BlockStmt{List: rightStatements}})
	if r.untypedBoolean(expression) {
		return statements, &ast.BinaryExpr{X: result, Op: token.EQL, Y: boolean(true)}
	}
	return statements, result
}

func (r *rewrite) selectorExpression(e *ast.SelectorExpr) ([]ast.Stmt, ast.Expr) {
	output := *e
	if selection := r.input.source.inventory.Checked.TypesInfo.Selections[e]; selection != nil &&
		selection.Kind() == types.MethodVal {
		signature := selection.Obj().Type().(*types.Signature)
		_, pointerReceiver := signature.Recv().Type().Underlying().(*types.Pointer)
		_, pointerValue := r.input.source.inventory.Checked.TypesInfo.TypeOf(e.X).Underlying().(*types.Pointer)
		if pointerReceiver && !pointerValue && r.input.source.inventory.Checked.TypesInfo.Types[e.X].Addressable() {
			statements, receiver := r.address(e.X)
			output.X = receiver
			return statements, &output
		}
	}
	statementsEvaluation := r.expression(e.X)
	statements, values := statementsEvaluation.statements, statementsEvaluation.expressions()
	output.X = values[0]
	return statements, &output
}

func (r *rewrite) indexExpression(e *ast.IndexExpr) ([]ast.Stmt, ast.Expr) {
	output := *e
	if storageKinds(r.input.source.inventory.Checked.TypesInfo.TypeOf(e.X)) == arrayStorage|otherStorage &&
		r.input.source.inventory.Checked.TypesInfo.Types[e.X].Addressable() {
		statements, base := r.genericBase(e.X)
		indexStatements, index := r.capture(e.Index)
		statements = append(statements, indexStatements...)
		output.X = base
		output.Index = index
		return statements, &output
	}
	if r.addressableArray(e.X) && r.contains(e.Index) {
		statements, base := r.arrayBase(e.X)
		indexEvaluation := r.expression(e.Index)
		indexStatements, index := indexEvaluation.statements, indexEvaluation.expressions()
		statements = append(statements, indexStatements...)
		output.X = base
		output.Index = index[0]
		return statements, &output
	}
	statementsEvaluation := r.operands([]ast.Expr{e.X, e.Index})
	statements, values := statementsEvaluation.statements, statementsEvaluation.expressions()
	output.X = values[0]
	output.Index = values[1]
	return statements, &output
}

func (r *rewrite) sliceExpression(e *ast.SliceExpr) ([]ast.Stmt, ast.Expr) {
	var statements []ast.Stmt
	output := *e
	input := []ast.Expr{e.X}
	array := r.addressableArray(e.X)
	mixed := storageKinds(r.input.source.inventory.Checked.TypesInfo.TypeOf(e.X)) == arrayStorage|
		otherStorage && r.input.source.inventory.Checked.TypesInfo.Types[e.X].Addressable()
	if mixed {
		statements, output.X = r.genericBase(e.X)
		input = nil
	}
	if array {
		statements, output.X = r.arrayBase(e.X)
		input = nil
	}
	for _, bound := range []ast.Expr{e.Low, e.High, e.Max} {
		if bound != nil {
			input = append(input, bound)
		}
	}
	operandEvaluation := r.operands(input)
	operandStatements, values := operandEvaluation.statements, operandEvaluation.expressions()
	statements = append(statements, operandStatements...)
	if mixed {
		for i, value := range values {
			boundEvaluation := r.save(r.value(value))
			boundStatements, savedBounds := boundEvaluation.statements, boundEvaluation.expressions()
			statements = append(statements, boundStatements...)
			values[i] = savedBounds[0]
		}
	}
	boundIndex := 0
	if !array && !mixed {
		output.X = values[0]
		boundIndex = 1
	}
	for _, bound := range []*ast.Expr{&output.Low, &output.High, &output.Max} {
		if *bound != nil {
			*bound = values[boundIndex]
			boundIndex++
		}
	}
	return statements, &output
}

func (r *rewrite) compositeExpression(e *ast.CompositeLit) ([]ast.Stmt, ast.Expr) {
	var statements []ast.Stmt
	output := *e
	var input []ast.Expr
	for _, element := range e.Elts {
		if pair, ok := element.(*ast.KeyValueExpr); ok {
			if tv := r.input.source.inventory.Checked.TypesInfo.Types[pair.Key]; tv.IsValue() {
				input = append(input, pair.Key)
			}
			input = append(input, pair.Value)
		} else {
			input = append(input, element)
		}
	}
	statementsEvaluation := r.operands(input)
	statements, values := statementsEvaluation.statements, statementsEvaluation.expressions()
	output.Elts = make([]ast.Expr, len(e.Elts))
	operandIndex := 0
	for elementIndex, element := range e.Elts {
		if pair, ok := element.(*ast.KeyValueExpr); ok {
			keyValue := *pair
			if tv := r.input.source.inventory.Checked.TypesInfo.Types[pair.Key]; tv.IsValue() {
				keyValue.Key = values[operandIndex]
				operandIndex++
			}
			keyValue.Value = values[operandIndex]
			operandIndex++
			output.Elts[elementIndex] = &keyValue
		} else {
			output.Elts[elementIndex] = values[operandIndex]
			operandIndex++
		}
	}
	return statements, &output
}
