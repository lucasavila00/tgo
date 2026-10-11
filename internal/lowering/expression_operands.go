package lowering

import (
	"github.com/lucasavila00/tgo/internal/checker/types"
	"go/ast"
	"go/token"
)

func (r *rewrite) binaryExpression(e *ast.BinaryExpr) ([]ast.Stmt, ast.Expr) {
	var before []ast.Stmt
	var result ast.Expr
	copy := *e
	if e.Op == token.LAND || e.Op == token.LOR {
		leftBefore, left := r.expression(e.X)
		rightBefore, right := r.expression(e.Y)
		before = leftBefore
		copy.X = left[0]
		copy.Y = right[0]
		if len(rightBefore) == 0 {
			result = &copy
			return before, result
		}
		value := r.name()
		before = append(before, variable(value, r.temporaryType(r.typeOf(e), e), nil))
		before = append(before, &ast.AssignStmt{Lhs: []ast.Expr{value}, Tok: token.ASSIGN, Rhs: left})
		var condition ast.Expr = value
		if e.Op == token.LOR {
			condition = &ast.UnaryExpr{Op: token.NOT, X: value}
		}
		rightBefore = append(rightBefore, &ast.AssignStmt{Lhs: []ast.Expr{value}, Tok: token.ASSIGN, Rhs: right})
		before = append(before, &ast.IfStmt{Cond: condition, Body: &ast.BlockStmt{List: rightBefore}})
		result = value
		if r.untypedBoolean(e) {
			result = &ast.BinaryExpr{X: value, Op: token.EQL, Y: boolean(true)}
		}
	} else {
		var values []ast.Expr
		before, values = r.operands([]ast.Expr{e.X, e.Y})
		copy.X = values[0]
		copy.Y = values[1]
		result = &copy
	}
	return before, result
}

func (r *rewrite) selectorExpression(e *ast.SelectorExpr) ([]ast.Stmt, ast.Expr) {
	var before []ast.Stmt
	var result ast.Expr
	copy := *e
	if selection := r.source.checked.TypesInfo.Selections[e]; selection != nil && selection.Kind() == types.MethodVal {
		signature := selection.Obj().Type().(*types.Signature)
		_, pointerReceiver := signature.Recv().Type().Underlying().(*types.Pointer)
		_, pointerValue := r.source.checked.TypesInfo.TypeOf(e.X).Underlying().(*types.Pointer)
		if pointerReceiver && !pointerValue && r.source.checked.TypesInfo.Types[e.X].Addressable() {
			before, copy.X = r.address(e.X)
			result = &copy
			return before, result
		}
	}
	var values []ast.Expr
	before, values = r.expression(e.X)
	copy.X = values[0]
	result = &copy
	return before, result
}

func (r *rewrite) indexExpression(e *ast.IndexExpr) ([]ast.Stmt, ast.Expr) {
	var before []ast.Stmt
	var result ast.Expr
	copy := *e
	if storageKinds(r.source.checked.TypesInfo.TypeOf(e.X)) == arrayStorage|otherStorage && r.source.checked.TypesInfo.Types[e.X].Addressable() {
		before, copy.X = r.genericBase(e.X)
		setup, index := r.capture(e.Index)
		before = append(before, setup...)
		copy.Index = index
		result = &copy
		return before, result
	}
	if r.addressableArray(e.X) && r.contains(e.Index) {
		var base ast.Expr
		before, base = r.arrayBase(e.X)
		setup, index := r.expression(e.Index)
		before = append(before, setup...)
		copy.X = base
		copy.Index = index[0]
		result = &copy
		return before, result
	}
	var values []ast.Expr
	before, values = r.operands([]ast.Expr{e.X, e.Index})
	copy.X = values[0]
	copy.Index = values[1]
	result = &copy
	return before, result
}

func (r *rewrite) sliceExpression(e *ast.SliceExpr) ([]ast.Stmt, ast.Expr) {
	var before []ast.Stmt
	var result ast.Expr
	copy := *e
	input := []ast.Expr{e.X}
	array := r.addressableArray(e.X)
	mixed := storageKinds(r.source.checked.TypesInfo.TypeOf(e.X)) == arrayStorage|otherStorage && r.source.checked.TypesInfo.Types[e.X].Addressable()
	if mixed {
		before, copy.X = r.genericBase(e.X)
		input = nil
	}
	if array {
		before, copy.X = r.arrayBase(e.X)
		input = nil
	}
	for _, bound := range []ast.Expr{e.Low, e.High, e.Max} {
		if bound != nil {
			input = append(input, bound)
		}
	}
	var values []ast.Expr
	setup, values := r.operands(input)
	before = append(before, setup...)
	if mixed {
		for i, value := range values {
			saved, refs := r.save(value)
			before = append(before, saved...)
			values[i] = refs[0]
		}
	}
	i := 0
	if !array && !mixed {
		copy.X = values[0]
		i = 1
	}
	for _, bound := range []*ast.Expr{&copy.Low, &copy.High, &copy.Max} {
		if *bound != nil {
			*bound = values[i]
			i++
		}
	}
	result = &copy
	return before, result
}

func (r *rewrite) compositeExpression(e *ast.CompositeLit) ([]ast.Stmt, ast.Expr) {
	var before []ast.Stmt
	var result ast.Expr
	copy := *e
	var input []ast.Expr
	for _, element := range e.Elts {
		if pair, ok := element.(*ast.KeyValueExpr); ok {
			if tv := r.source.checked.TypesInfo.Types[pair.Key]; tv.IsValue() {
				input = append(input, pair.Key)
			}
			input = append(input, pair.Value)
		} else {
			input = append(input, element)
		}
	}
	var values []ast.Expr
	before, values = r.operands(input)
	copy.Elts = make([]ast.Expr, len(e.Elts))
	operandIndex := 0
	for elementIndex, element := range e.Elts {
		if pair, ok := element.(*ast.KeyValueExpr); ok {
			keyValue := *pair
			if tv := r.source.checked.TypesInfo.Types[pair.Key]; tv.IsValue() {
				keyValue.Key = values[operandIndex]
				operandIndex++
			}
			keyValue.Value = values[operandIndex]
			operandIndex++
			copy.Elts[elementIndex] = &keyValue
		} else {
			copy.Elts[elementIndex] = values[operandIndex]
			operandIndex++
		}
	}
	result = &copy
	return before, result
}
