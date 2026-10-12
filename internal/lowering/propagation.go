package lowering

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"github.com/lucasavila00/tgo/internal/checker/types"
)

func (r *rewrite) propagationExpression(marker *ast.CallExpr, operation Propagation,
	discardSuccess bool) ([]ast.Stmt, []ast.Expr) {
	call := marker.Args[0]
	statements, callValues := r.expression(call)
	saveStatements, savedValues := r.saveResults(callValues[0], discardSuccess, true)
	statements = append(statements, saveStatements...)
	if len(savedValues) == 0 {
		panic("propagation requires an error result")
	}

	returnErrorType := r.errorReturnType()
	if _, ok := returnErrorType.Underlying().(*types.Interface); !ok || !types.AssignableTo(returnErrorType,
		types.Universe.Lookup("error").Type()) {
		panic("propagation requires a final error interface return")
	}
	zeroResults := r.failureResults()

	callError := savedValues[len(savedValues)-1]
	failureStatements, failed := r.callFailure(callError, call)
	statements = append(statements, failureStatements...)
	errorValue := r.propagatedError(callError, returnErrorType, operation, call)
	results := append(zeroResults, errorValue)
	failure := &ast.IfStmt{Cond: failed, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: results}}}}
	statements = append(statements, failure)
	r.completePropagation(marker)
	if discardSuccess {
		return statements, nil
	}
	return statements, savedValues[:len(savedValues)-1]
}

func (r *rewrite) propagatedError(callError ast.Expr, returnErrorType types.Type,
	operation Propagation, call ast.Expr) ast.Expr {
	if !operation.Wrap {
		return callError
	}
	if !types.AssignableTo(types.Universe.Lookup("error").Type(), returnErrorType) {
		panic("wrapped errors do not fit the function return type")
	}
	formatPackage := r.formatPackage(call)
	message := strings.ReplaceAll(operation.CallName, "%", "%%") + ": %w"
	return &ast.CallExpr{
		Fun:  &ast.SelectorExpr{X: formatPackage, Sel: ast.NewIdent("Errorf")},
		Args: []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(message)}, callError},
	}
}

// callFailure checks the call result before conversion to the caller's return type.
func (r *rewrite) callFailure(value ast.Expr, call ast.Expr) ([]ast.Stmt, ast.Expr) {
	typ := r.typeOf(value)
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
		scope := r.input.source.inventory.Checked.Types.Scope().Innermost(call.Pos())
		_, resolved := scope.LookupParent("nil", call.Pos())
		if resolved == types.Universe.Lookup("nil") {
			return nil, &ast.BinaryExpr{X: value, Op: token.NEQ, Y: ast.NewIdent("nil")}
		}
	}
	if generic || nilOnly {
		return r.reflectedFailure(value, call)
	}
	switch typ.Underlying().(type) {
	case *types.Pointer, *types.Chan, *types.Interface:
		zero := r.nameFor("zero")
		return []ast.Stmt{variable(zero, r.temporaryType(typ, call), nil)}, &ast.BinaryExpr{X: value,
			Op: token.NEQ, Y: zero}
	default:
		return nil, boolean(true)
	}
}

func (r *rewrite) reflectedFailure(value ast.Expr, call ast.Expr) ([]ast.Stmt, ast.Expr) {
	reflectPackage := r.reflectPackage()
	reflected := r.nameFor("reflected")
	failed := r.nameFor("failed")
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
		variable(reflected, &ast.SelectorExpr{X: reflectPackage, Sel: ast.NewIdent("Value")}, reflectedValue),
		variable(failed, r.temporaryType(types.Typ[types.Bool], call), boolean(true)),
		checkNil,
	}
	return statements, failed
}
