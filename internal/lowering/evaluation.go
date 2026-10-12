package lowering

import (
	"go/ast"
	"go/token"

	"github.com/lucasavila00/tgo/internal/checker/types"
)

func (r *rewrite) recordOrigin(expression, original ast.Expr) {
	value := r.value(expression)
	value.Original = original
	if !value.Temporary {
		value.Type = r.input.source.inventory.Checked.TypesInfo.TypeOf(original)
	}
	r.recordValue(value)
}

func (r *rewrite) recordTemporary(expression ast.Expr, typ types.Type, original ast.Expr) {
	r.recordValue(expressionValue{Expression: expression, Type: typ, Original: original, Temporary: true})
}

func (r *rewrite) recordValue(value expressionValue) {
	if r.functionState.values == nil {
		r.functionState.values = make(map[ast.Expr]expressionValue)
	}
	r.functionState.values[value.Expression] = value
}

func (r *rewrite) save(value expressionValue) expressionResult {
	return r.saveResults(value, false, false)
}

func (r *rewrite) saveResults(value expressionValue, discardSuccess, errorResult bool) expressionResult {
	expr := value.Expression
	original := value.Original
	count := r.resultCount(original)
	if value.Temporary {
		count = 1
	}
	if literal := r.implicitMapKey(value); literal != nil {
		return r.materializeKey(expr.(*ast.CompositeLit), literal)
	}
	typ := value.Type
	if !value.Temporary {
		expr = r.contextLiteral(expr, original)
	}
	if count == 0 {
		return r.result([]ast.Stmt{&ast.ExprStmt{X: expr}}, nil)
	}
	refs := r.savedReferences(count, discardSuccess, errorResult)
	var statements []ast.Stmt
	for i, ref := range refs {
		if ref.(*ast.Ident).Name == "_" {
			continue
		}
		resultType := typ
		if tuple, ok := typ.(*types.Tuple); ok {
			resultType = tuple.At(i).Type()
		} else if i == 1 {
			resultType = types.Typ[types.Bool]
		}
		var value ast.Expr
		if count == 1 {
			value = expr
		}
		statements = append(statements, variable(ref.(*ast.Ident), r.temporaryType(resultType, original), value))
		r.recordTemporary(ref, types.Default(resultType), original)
	}
	if count != 1 {
		statements = append(statements, &ast.AssignStmt{Lhs: refs, Tok: token.ASSIGN, Rhs: []ast.Expr{expr}})
	}
	if count == 1 && r.untypedBoolean(original) {
		comparison := &ast.BinaryExpr{X: refs[0], Op: token.EQL, Y: boolean(true)}
		r.recordValue(expressionValue{Expression: comparison, Type: types.Typ[types.UntypedBool], Original: original})
		return r.result(statements, []ast.Expr{comparison})
	}
	return r.result(statements, refs)
}

func (r *rewrite) resultCount(expr ast.Expr) int {
	if tuple, ok := r.input.source.inventory.Checked.TypesInfo.TypeOf(expr).(*types.Tuple); ok {
		return tuple.Len()
	}
	if r.input.source.inventory.Checked.TypesInfo.Types[expr].HasOk() {
		var node ast.Node = expr
		for {
			if parenthesized, ok := r.input.source.inventory.Parents[node].(*ast.ParenExpr); ok {
				node = parenthesized
			} else {
				break
			}
		}
		switch parent := r.input.source.inventory.Parents[node].(type) {
		case *ast.AssignStmt:
			if len(parent.Rhs) == 1 && len(parent.Lhs) == 2 {
				return 2
			}
		case *ast.ValueSpec:
			if len(parent.Values) == 1 && len(parent.Names) == 2 {
				return 2
			}
		}
	}
	return 1
}

// operands saves earlier values before a later operand adds statements.
func (r *rewrite) operands(input []ast.Expr) expressionResult {
	var result expressionResult
	for _, expression := range input {
		operand := r.expression(expression)
		if len(operand.Statements) != 0 {
			saved := r.savePreviousOperands(result.Values)
			result.Values = saved.Values
			result.Statements = append(result.Statements, saved.Statements...)
			result.Statements = append(result.Statements, operand.Statements...)
		}
		result.Values = append(result.Values, operand.Values...)
	}
	return result
}

func (r *rewrite) savePreviousOperands(values []expressionValue) expressionResult {
	result := expressionResult{Values: values}
	for i, value := range values {
		if !r.operandNeedsSaving(value.Expression) {
			continue
		}
		saved := r.save(value)
		result.Statements = append(result.Statements, saved.Statements...)
		result.Values[i] = saved.Values[0]
	}
	return result
}

func (r *rewrite) operandNeedsSaving(value ast.Expr) bool {
	if r.functionName(value) {
		return false
	}
	if typeAndValue, ok := r.input.source.inventory.Checked.TypesInfo.Types[value]; ok {
		if typeAndValue.Value != nil || typeAndValue.IsType() || typeAndValue.IsBuiltin() || typeAndValue.IsNil() {
			return false
		}
	}
	if _, identifier := value.(*ast.Ident); identifier && r.input.source.inventory.Checked.TypesInfo.TypeOf(value) == nil {
		return false
	}
	return true
}

func (r *rewrite) savedReferences(count int, discardSuccess, errorResult bool) []ast.Expr {
	refs := make([]ast.Expr, count)
	for i := range refs {
		prefix := "value"
		if errorResult && i == len(refs)-1 {
			prefix = "error"
		} else if discardSuccess {
			refs[i] = ast.NewIdent("_")
			continue
		}
		refs[i] = r.nameFor(prefix)
	}
	return refs
}

func (r *rewrite) implicitMapKey(value expressionValue) *ast.CompositeLit {
	original := value.Original
	literal, ok := original.(*ast.CompositeLit)
	if !ok || literal.Type != nil || value.Temporary {
		return nil
	}
	pair, ok := r.input.source.inventory.Parents[original].(*ast.KeyValueExpr)
	if !ok || pair.Key != original {
		return nil
	}
	if _, pointer := r.input.source.inventory.Checked.TypesInfo.TypeOf(original).Underlying().(*types.Pointer); pointer {
		return nil
	}
	return literal
}
