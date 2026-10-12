package propagation

import (
	"github.com/lucasavila00/tgo/internal/lowering/shared"
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"github.com/lucasavila00/tgo/internal/checker/types"
)

// Expression evaluates a marked call and emits its failure return.
func Expression(r *shared.Context, marker *ast.CallExpr, operation shared.Propagation,
	discardSuccess bool) shared.Result {
	call := marker.Args[0]
	evaluation := r.Expression(call)
	saved := r.SaveResults(evaluation.Values[0], discardSuccess, true)
	statements := append(evaluation.Statements, saved.Statements...)
	if len(saved.Values) == 0 {
		panic("propagation requires an error result")
	}

	returnErrorType := r.ErrorReturnType()
	if _, ok := returnErrorType.Underlying().(*types.Interface); !ok || !types.AssignableTo(returnErrorType,
		types.Universe.Lookup("error").Type()) {
		panic("propagation requires a final error interface return")
	}
	zeroResults := r.FailureResults()

	callError := saved.Values[len(saved.Values)-1]
	failureStatements, failed := callFailure(r, callError, call)
	statements = append(statements, failureStatements...)
	errorValue := propagatedError(r, callError.Expression, returnErrorType, operation, call)
	results := append(zeroResults, errorValue)
	failure := &ast.IfStmt{Cond: failed, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: results}}}}
	statements = append(statements, failure)
	r.CompletePropagation(marker)
	if discardSuccess {
		return shared.Result{Statements: statements}
	}
	return shared.Result{Statements: statements, Values: saved.Values[:len(saved.Values)-1]}
}

func propagatedError(r *shared.Context, callError ast.Expr, returnErrorType types.Type,
	operation shared.Propagation, call ast.Expr) ast.Expr {
	if !operation.Wrap {
		return callError
	}
	if !types.AssignableTo(types.Universe.Lookup("error").Type(), returnErrorType) {
		panic("wrapped errors do not fit the function return type")
	}
	formatPackage := r.FormatPackage(call)
	message := strings.ReplaceAll(operation.CallName, "%", "%%") + ": %w"
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
		scope := r.Inventory().Checked.Types.Scope().Innermost(call.Pos())
		_, resolved := scope.LookupParent("nil", call.Pos())
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
