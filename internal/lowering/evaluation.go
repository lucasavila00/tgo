package lowering

import (
	"go/ast"
	"go/token"

	"github.com/lucasavila00/tgo/internal/checker/types"
)

func (r *rewrite) recordOrigin(expression, original ast.Expr) {
	if r.origins == nil {
		r.origins = make(map[ast.Expr]ast.Expr)
	}
	r.origins[expression] = original
}

func (r *rewrite) recordTemporary(expression ast.Expr, typ types.Type, original ast.Expr) {
	if r.temporaryTypes == nil {
		r.temporaryTypes = make(map[ast.Expr]types.Type)
	}
	r.temporaryTypes[expression] = typ
	r.recordOrigin(expression, original)
}

func (r *rewrite) save(expr ast.Expr) ([]ast.Stmt, []ast.Expr) {
	original := expr
	if origin := r.origins[expr]; origin != nil {
		original = origin
	}
	count := r.resultCount(original)
	if r.temporaryTypes[expr] != nil {
		count = 1
	}
	if literal, ok := original.(*ast.CompositeLit); ok && literal.Type == nil && r.temporaryTypes[expr] == nil {
		if pair, ok := r.source.inventory.Parents[original].(*ast.KeyValueExpr); ok && pair.Key == original {
			if _, pointer := r.source.inventory.Checked.TypesInfo.TypeOf(original).Underlying().(*types.Pointer); !pointer {
				return r.materializeKey(expr.(*ast.CompositeLit), literal)
			}
		}
	}
	typ := r.typeOf(expr)
	if r.temporaryTypes[expr] == nil {
		expr = r.contextLiteral(expr, original)
	}
	if count == 0 {
		return []ast.Stmt{&ast.ExprStmt{X: expr}}, nil
	}
	refs := make([]ast.Expr, count)
	for i := range refs {
		refs[i] = r.name()
	}
	var statements []ast.Stmt
	for i, ref := range refs {
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
		return statements, []ast.Expr{&ast.BinaryExpr{X: refs[0], Op: token.EQL, Y: boolean(true)}}
	}
	return statements, refs
}

func (r *rewrite) resultCount(expr ast.Expr) int {
	if tuple, ok := r.source.inventory.Checked.TypesInfo.TypeOf(expr).(*types.Tuple); ok {
		return tuple.Len()
	}
	if r.source.inventory.Checked.TypesInfo.Types[expr].HasOk() {
		var node ast.Node = expr
		for {
			if parenthesized, ok := r.source.inventory.Parents[node].(*ast.ParenExpr); ok {
				node = parenthesized
			} else {
				break
			}
		}
		switch parent := r.source.inventory.Parents[node].(type) {
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
func (r *rewrite) operands(input []ast.Expr) ([]ast.Stmt, []ast.Expr) {
	var statements []ast.Stmt
	var values []ast.Expr
	for _, expr := range input {
		operandStatements, operandValues := r.expression(expr)
		if len(operandStatements) != 0 {
			var saveStatements []ast.Stmt
			saveStatements, values = r.savePreviousOperands(values)
			statements = append(statements, saveStatements...)
			statements = append(statements, operandStatements...)
		}
		values = append(values, operandValues...)
	}
	return statements, values
}

func (r *rewrite) savePreviousOperands(values []ast.Expr) ([]ast.Stmt, []ast.Expr) {
	var statements []ast.Stmt
	for i, value := range values {
		if !r.operandNeedsSaving(value) {
			continue
		}
		saveStatements, savedValues := r.save(value)
		statements = append(statements, saveStatements...)
		values[i] = savedValues[0]
	}
	return statements, values
}

func (r *rewrite) operandNeedsSaving(value ast.Expr) bool {
	if r.functionName(value) {
		return false
	}
	if typeAndValue, ok := r.source.inventory.Checked.TypesInfo.Types[value]; ok {
		if typeAndValue.Value != nil || typeAndValue.IsType() || typeAndValue.IsBuiltin() || typeAndValue.IsNil() {
			return false
		}
	}
	if _, identifier := value.(*ast.Ident); identifier && r.source.inventory.Checked.TypesInfo.TypeOf(value) == nil {
		return false
	}
	return true
}
