package shared

import (
	"github.com/lucasavila00/tgo/internal/ast"
	"github.com/lucasavila00/tgo/internal/checker/types"
	"go/token"
)

// binaryExpression lowers binary operands in order and delegates short-circuit operators to branch lowering.
func (r *Context) binaryExpression(e *ast.BinaryExpr) ([]ast.Stmt, ast.Expr) {
	if e.Op == token.LAND || e.Op == token.LOR {
		return r.logicalExpression(e)
	}
	statementsEvaluation := r.Operands([]ast.Expr{e.X, e.Y})
	statements, values := statementsEvaluation.Statements, statementsEvaluation.Expressions()
	output := *e
	output.X = values[0]
	output.Y = values[1]
	return statements, &output
}

// logicalExpression puts the right operand's added statements inside the branch that evaluates it.
func (r *Context) logicalExpression(expression *ast.BinaryExpr) ([]ast.Stmt, ast.Expr) {
	leftEvaluation := r.Expression(expression.X)
	leftStatements, leftValues := leftEvaluation.Statements, leftEvaluation.Expressions()
	rightEvaluation := r.Expression(expression.Y)
	rightStatements, rightValues := rightEvaluation.Statements, rightEvaluation.Expressions()

	output := *expression
	output.X = leftValues[0]
	output.Y = rightValues[0]
	if len(rightStatements) == 0 {
		return leftStatements, &output
	}

	result := r.Name()
	statements := append(leftStatements, Variable(result, r.TemporaryType(r.TypeOf(expression), expression), nil))
	statements = append(statements, &ast.AssignStmt{Lhs: []ast.Expr{result}, Tok: token.ASSIGN, Rhs: leftValues})
	condition := ast.Expr(result)
	if expression.Op == token.LOR {
		condition = &ast.UnaryExpr{Op: token.NOT, X: result}
	}
	rightStatements = append(rightStatements, &ast.AssignStmt{Lhs: []ast.Expr{result}, Tok: token.ASSIGN,
		Rhs: rightValues})
	statements = append(statements, &ast.IfStmt{Cond: condition, Body: &ast.BlockStmt{List: rightStatements}})
	if r.untypedBoolean(expression) {
		return statements, &ast.BinaryExpr{X: result, Op: token.EQL, Y: Boolean(true)}
	}
	return statements, result
}

// selectorExpression lowers a selector receiver while retaining storage for an implicit pointer method receiver.
func (r *Context) selectorExpression(e *ast.SelectorExpr) ([]ast.Stmt, ast.Expr) {
	output := *e
	if selection := r.input.Inventory.Checked.TypesInfo.Selections[e]; selection != nil &&
		selection.Kind() == types.MethodVal {
		signature := selection.Obj().Type().(*types.Signature)
		_, pointerReceiver := signature.Recv().Type().Underlying().(*types.Pointer)
		_, pointerValue := r.input.Inventory.Checked.TypesInfo.TypeOf(e.X).Underlying().(*types.Pointer)
		if pointerReceiver && !pointerValue && r.input.Inventory.Checked.TypesInfo.Types[e.X].Addressable() {
			statements, receiver := r.address(e.X)
			output.X = receiver
			return statements, &output
		}
	}
	statementsEvaluation := r.Expression(e.X)
	statements, values := statementsEvaluation.Statements, statementsEvaluation.Expressions()
	output.X = values[0]
	return statements, &output
}

// indexExpression lowers an index in operand order, retaining array storage when later work changes its contents.
func (r *Context) indexExpression(e *ast.IndexExpr) ([]ast.Stmt, ast.Expr) {
	output := *e
	if storageKinds(r.input.Inventory.Checked.TypesInfo.TypeOf(e.X)) == arrayStorage|otherStorage &&
		r.input.Inventory.Checked.TypesInfo.Types[e.X].Addressable() {
		statements, base := r.genericBase(e.X)
		indexStatements, index := r.Capture(e.Index)
		statements = append(statements, indexStatements...)
		output.X = base
		output.Index = index
		return statements, &output
	}
	if r.addressableArray(e.X) && r.Contains(e.Index) {
		statements, base := r.arrayBase(e.X)
		indexEvaluation := r.Expression(e.Index)
		indexStatements, index := indexEvaluation.Statements, indexEvaluation.Expressions()
		statements = append(statements, indexStatements...)
		output.X = base
		output.Index = index[0]
		return statements, &output
	}
	statementsEvaluation := r.Operands([]ast.Expr{e.X, e.Index})
	statements, values := statementsEvaluation.Statements, statementsEvaluation.Expressions()
	output.X = values[0]
	output.Index = values[1]
	return statements, &output
}

// sliceExpression lowers slice bounds in order while retaining array storage or a saved slice header.
func (r *Context) sliceExpression(e *ast.SliceExpr) ([]ast.Stmt, ast.Expr) {
	var statements []ast.Stmt
	output := *e
	input := []ast.Expr{e.X}
	array := r.addressableArray(e.X)
	mixed := storageKinds(r.input.Inventory.Checked.TypesInfo.TypeOf(e.X)) == arrayStorage|
		otherStorage && r.input.Inventory.Checked.TypesInfo.Types[e.X].Addressable()
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
	operandEvaluation := r.Operands(input)
	operandStatements, values := operandEvaluation.Statements, operandEvaluation.Expressions()
	statements = append(statements, operandStatements...)
	if mixed {
		for i, value := range values {
			boundEvaluation := r.Save(r.ValueOf(value))
			boundStatements, savedBounds := boundEvaluation.Statements, boundEvaluation.Expressions()
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

// compositeExpression lowers literal keys and values in order without treating field names as evaluated keys.
func (r *Context) compositeExpression(e *ast.CompositeLit) ([]ast.Stmt, ast.Expr) {
	var statements []ast.Stmt
	output := *e
	var input []ast.Expr
	for _, element := range e.Elts {
		if pair, ok := element.(*ast.KeyValueExpr); ok {
			if tv := r.input.Inventory.Checked.TypesInfo.Types[pair.Key]; tv.IsValue() {
				input = append(input, pair.Key)
			}
			input = append(input, pair.Value)
		} else {
			input = append(input, element)
		}
	}
	statementsEvaluation := r.Operands(input)
	statements, values := statementsEvaluation.Statements, statementsEvaluation.Expressions()
	output.Elts = make([]ast.Expr, len(e.Elts))
	operandIndex := 0
	for elementIndex, element := range e.Elts {
		if pair, ok := element.(*ast.KeyValueExpr); ok {
			keyValue := *pair
			if tv := r.input.Inventory.Checked.TypesInfo.Types[pair.Key]; tv.IsValue() {
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
