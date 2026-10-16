package propagation

import (
	"github.com/lucasavila00/tgo/internal/ast"
	"github.com/lucasavila00/tgo/internal/lowering/shared"
	"go/token"
	"strconv"
	"strings"

	"github.com/lucasavila00/tgo/internal/checker/types"
)

// Expression evaluates the immediate operand and returns its final error on failure.
func Expression(r *shared.Context, expression *ast.PropagateExpr, discardSuccess bool) shared.Result {
	call := expression.X
	evaluation := r.Expression(call)
	saved := saveOperandResults(r, evaluation.Values, discardSuccess)
	statements := append(evaluation.Statements, saved.Statements...)
	if len(saved.Values) == 0 {
		panic("propagation requires an error result")
	}

	returnErrorType := r.ErrorReturnType()
	if _, ok := returnErrorType.Underlying().(*types.Interface); !ok || !types.AssignableTo(returnErrorType,
		types.Universe.Lookup("error").Type()) {
		panic("propagation requires a final error interface return")
	}
	zeroResults := r.PrepareFailureResults()

	callError := saved.Values[len(saved.Values)-1]
	failureStatements, failed := callFailure(r, callError, call)
	statements = append(statements, failureStatements...)
	errorValue := propagatedError(r, callError.Expression, returnErrorType, expression, call)
	results := append(zeroResults, errorValue)
	failure := &ast.IfStmt{Cond: failed, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: results}}}}
	statements = append(statements, failure)
	r.CompletePropagation(expression)
	if discardSuccess {
		return shared.Result{Statements: statements}
	}
	return shared.Result{Statements: statements, Values: saved.Values[:len(saved.Values)-1]}
}

// saveOperandResults handles a tuple call or results already saved by nested propagation.
func saveOperandResults(r *shared.Context, values []shared.Value, discardSuccess bool) shared.Result {
	if len(values) == 1 {
		return r.SaveResults(values[0], discardSuccess, true)
	}
	result := shared.Result{Values: append([]shared.Value(nil), values...)}
	for index, value := range values {
		if discardSuccess && index != len(values)-1 {
			discard := &ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent("_")}, Tok: token.ASSIGN, Rhs: []ast.Expr{value.Expression}}
			result.Statements = append(result.Statements, discard)
			continue
		}
		if !value.Temporary {
			saved := r.Save(value)
			result.Statements = append(result.Statements, saved.Statements...)
			result.Values[index] = saved.Values[0]
		}
	}
	return result
}

// propagatedError builds the failure value, adding the source call name only when context was requested.
func propagatedError(r *shared.Context, callError ast.Expr, returnErrorType types.Type,
	operation *ast.PropagateExpr, call ast.Expr) ast.Expr {
	if !operation.Wrap {
		return callError
	}
	if !types.AssignableTo(types.Universe.Lookup("error").Type(), returnErrorType) {
		panic("wrapped errors do not fit the function return type")
	}
	formatPackage := r.FormatPackage(call)
	message := strings.ReplaceAll(r.Inventory().CallName(operation), "%", "%%") + ": %w"
	return &ast.CallExpr{
		Fun:  &ast.SelectorExpr{X: formatPackage, Sel: ast.NewIdent("Errorf")},
		Args: []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(message)}, callError},
	}
}

// callFailure checks the call result before conversion to the caller's return type.
func callFailure(r *shared.Context, result shared.Value, call ast.Expr) ([]ast.Stmt, ast.Expr) {
	value := result.Expression
	typ := result.Type
	_, generic := types.Unalias(typ).(*types.TypeParam)
	nilOnly := false
	nilCapable := false
	switch typ.Underlying().(type) {
	case *types.Slice, *types.Map, *types.Signature:
		nilOnly = true
		nilCapable = true
	case *types.Pointer, *types.Chan:
		nilCapable = true
	case *types.Interface:
		nilCapable = !generic
	}
	if generic && types.AssignableTo(types.Typ[types.UntypedNil], typ) {
		nilCapable = true
	}
	if nilCapable {
		resolved := r.VisibleBinding(call, "nil")
		if resolved == types.Universe.Lookup("nil") {
			return nil, &ast.BinaryExpr{X: value, Op: token.NEQ, Y: ast.NewIdent("nil")}
		}
	}
	if generic || nilOnly {
		return reflectedFailure(r, value, call)
	}
	switch typ.Underlying().(type) {
	case *types.Pointer, *types.Chan, *types.Interface:
		zero := r.NameFor("zero")
		return []ast.Stmt{shared.Variable(zero, r.TemporaryType(typ, call), nil)}, &ast.BinaryExpr{X: value,
			Op: token.NEQ, Y: zero}
	default:
		return nil, shared.Boolean(true)
	}
}

// reflectedFailure builds a runtime nil check for error types that cannot use a direct comparison.
// It tests the saved result before conversion to the enclosing function's error type.
func reflectedFailure(r *shared.Context, value ast.Expr, call ast.Expr) ([]ast.Stmt, ast.Expr) {
	reflectPackage := r.ReflectPackage()
	reflected := r.NameFor("reflected")
	failed := r.NameFor("failed")
	valueAddress := &ast.UnaryExpr{Op: token.AND, X: value}
	valueOf := &ast.CallExpr{Fun: &ast.SelectorExpr{X: reflectPackage, Sel: ast.NewIdent("ValueOf")},
		Args: []ast.Expr{valueAddress}}
	reflectedValue := &ast.CallExpr{Fun: &ast.SelectorExpr{X: valueOf, Sel: ast.NewIdent("Elem")}}
	var nilCapableKinds []ast.Expr
	for _, kind := range []string{"Chan", "Func", "Interface", "Map", "Pointer", "Slice"} {
		nilCapableKinds = append(nilCapableKinds, &ast.SelectorExpr{X: reflectPackage, Sel: ast.NewIdent(kind)})
	}
	isNil := &ast.CallExpr{Fun: &ast.SelectorExpr{X: reflected, Sel: ast.NewIdent("IsNil")}}
	setFailure := &ast.AssignStmt{Lhs: []ast.Expr{failed}, Tok: token.ASSIGN,
		Rhs: []ast.Expr{&ast.UnaryExpr{Op: token.NOT, X: isNil}}}
	kind := &ast.CallExpr{Fun: &ast.SelectorExpr{X: reflected, Sel: ast.NewIdent("Kind")}}
	checkNil := &ast.SwitchStmt{Tag: kind, Body: &ast.BlockStmt{List: []ast.Stmt{
		&ast.CaseClause{List: nilCapableKinds, Body: []ast.Stmt{setFailure}},
	}}}
	statements := []ast.Stmt{
		shared.Variable(reflected, &ast.SelectorExpr{X: reflectPackage, Sel: ast.NewIdent("Value")}, reflectedValue),
		shared.Variable(failed, r.TemporaryType(types.Typ[types.Bool], call), shared.Boolean(true)),
		checkNil,
	}
	return statements, failed
}
