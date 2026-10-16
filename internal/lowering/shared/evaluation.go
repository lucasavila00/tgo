package shared

import (
	"github.com/lucasavila00/tgo/internal/ast"
	"go/token"

	"github.com/lucasavila00/tgo/internal/checker/types"
)

// recordOrigin records the source of a replacement expression and retains its checked type.
// It keeps the stored type when the replacement already names a temporary.
func (r *Context) recordOrigin(expression, original ast.Expr) {
	value := r.ValueOf(expression)
	value.Original = original
	if !value.Temporary {
		value.Type = r.input.Inventory.Checked.TypesInfo.TypeOf(original)
	}
	r.recordValue(value)
}

// recordTemporary records the checked type and source of a generated value in the current function.
func (r *Context) recordTemporary(expression ast.Expr, typ types.Type, original ast.Expr) {
	r.recordValue(Value{Expression: expression, Type: typ, Original: original, Temporary: true})
}

// recordValue stores value metadata for later type and origin queries in the current function.
func (r *Context) recordValue(value Value) {
	if r.functionState.values == nil {
		r.functionState.values = make(map[ast.Expr]Value)
	}
	r.functionState.values[value.Expression] = value
}

// Save evaluates a typed value once into fresh local storage.
func (r *Context) Save(value Value) Result {
	return r.SaveResults(value, false, false)
}

// SaveResults stores selected call results and retains their checked types.
func (r *Context) SaveResults(value Value, discardSuccess, errorResult bool) Result {
	expr := value.Expression
	original := value.Original
	count := 1
	if tuple, ok := value.Type.(*types.Tuple); ok {
		count = tuple.Len()
	} else if !value.Temporary && r.input.Inventory.Checked.TypesInfo.Types[original].HasOk() {
		count = r.resultCount(original)
	}
	if literal := r.implicitMapKey(value); literal != nil {
		return r.materializeKey(expr.(*ast.CompositeLit), literal)
	}
	typ := value.Type
	if !value.Temporary {
		expr = r.contextLiteral(expr, original)
	}
	if count == 0 {
		return r.ResultOf([]ast.Stmt{&ast.ExprStmt{X: expr}}, nil)
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
		statements = append(statements, Variable(ref.(*ast.Ident), r.TemporaryType(resultType, original), value))
		r.recordTemporary(ref, types.Default(resultType), original)
	}
	if count != 1 {
		statements = append(statements, &ast.AssignStmt{Lhs: refs, Tok: token.ASSIGN, Rhs: []ast.Expr{expr}})
	}
	if count == 1 && r.untypedBoolean(original) {
		comparison := &ast.BinaryExpr{X: refs[0], Op: token.EQL, Y: Boolean(true)}
		r.recordValue(Value{Expression: comparison, Type: types.Typ[types.UntypedBool], Original: original})
		return r.ResultOf(statements, []ast.Expr{comparison})
	}
	return r.ResultOf(statements, refs)
}

// resultCount counts tuple results or the two results of a comma-ok expression in its source context.
func (r *Context) resultCount(expr ast.Expr) int {
	if tuple, ok := r.input.Inventory.Checked.TypesInfo.TypeOf(expr).(*types.Tuple); ok {
		return tuple.Len()
	}
	if r.input.Inventory.Checked.TypesInfo.Types[expr].HasOk() {
		var node ast.Node = expr
		for {
			if parenthesized, ok := r.input.Inventory.Parents[node].(*ast.ParenExpr); ok {
				node = parenthesized
			} else {
				break
			}
		}
		switch parent := r.input.Inventory.Parents[node].(type) {
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

// Operands saves earlier values before a later operand adds statements.
func (r *Context) Operands(input []ast.Expr) Result {
	var result Result
	for _, expression := range input {
		operand := r.Expression(expression)
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

// savePreviousOperands saves earlier operands before later statements can change their values.
func (r *Context) savePreviousOperands(values []Value) Result {
	result := Result{Values: values}
	for i, value := range values {
		if !r.operandNeedsSaving(value.Expression) {
			continue
		}
		saved := r.Save(value)
		result.Statements = append(result.Statements, saved.Statements...)
		result.Values[i] = saved.Values[0]
	}
	return result
}

// operandNeedsSaving reports whether delaying this operand across added statements requires saved storage.
// Function names, constants, and type syntax do not require storage.
func (r *Context) operandNeedsSaving(value ast.Expr) bool {
	if r.functionName(value) {
		return false
	}
	if typeAndValue, ok := r.input.Inventory.Checked.TypesInfo.Types[value]; ok {
		if typeAndValue.Value != nil || typeAndValue.IsType() || typeAndValue.IsBuiltin() || typeAndValue.IsNil() {
			return false
		}
	}
	if _, identifier := value.(*ast.Ident); identifier && r.input.Inventory.Checked.TypesInfo.TypeOf(value) == nil {
		return false
	}
	return true
}

// savedReferences allocates fresh result names, using blank names for discarded success results.
func (r *Context) savedReferences(count int, discardSuccess, errorResult bool) []ast.Expr {
	refs := make([]ast.Expr, count)
	for i := range refs {
		prefix := "value"
		if errorResult && i == len(refs)-1 {
			prefix = "error"
		} else if discardSuccess {
			refs[i] = ast.NewIdent("_")
			continue
		}
		refs[i] = r.NameFor(prefix)
	}
	return refs
}

// implicitMapKey finds an omitted-type map key literal that needs field storage instead of early hashing.
func (r *Context) implicitMapKey(value Value) *ast.CompositeLit {
	original := value.Original
	literal, ok := original.(*ast.CompositeLit)
	if !ok || literal.Type != nil || value.Temporary {
		return nil
	}
	pair, ok := r.input.Inventory.Parents[original].(*ast.KeyValueExpr)
	if !ok || pair.Key != original {
		return nil
	}
	if _, pointer := r.input.Inventory.Checked.TypesInfo.TypeOf(original).Underlying().(*types.Pointer); pointer {
		return nil
	}
	return literal
}
