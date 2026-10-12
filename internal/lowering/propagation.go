package lowering

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"github.com/lucasavila00/tgo/internal/checker/types"
)

type returnFrame struct {
	function ast.Node
	typ      *ast.FuncType
	prepared bool
	prefix   []ast.Stmt
	values   []ast.Expr
}

func (r *rewrite) function(node ast.Node, typ *ast.FuncType, body *ast.BlockStmt) (*ast.FuncType,
	*ast.BlockStmt) {
	functionRewrite := &rewrite{
		input:         r.input,
		request:       r.request,
		file:          r.file,
		functionState: &functionState{frame: &returnFrame{function: node, typ: typ}, hasGoto: r.functionHasGoto(body)},
	}
	output := functionRewrite.block(body)
	if functionRewrite.functionState.frame.prepared {
		output.List = append(functionRewrite.functionState.frame.prefix, output.List...)
	}
	if functionRewrite.input.source.inventory.Checked.TypesInfo.TGoReturnBlocks[body] {
		output = functionRewrite.returnBlock(typ, output)
	}
	if aliases := functionRewrite.file.aliasStatements[node]; len(aliases) != 0 {
		output.List = append(aliases, output.List...)
	}
	return typ, output
}

// prepareReturns saves zero values for the results before the error.
func (r *rewrite) prepareReturns(original *ast.FuncType) []ast.Stmt {
	if original.Results == nil {
		panic("propagation requires an error return")
	}
	var prefix []ast.Stmt
	for i, field := range original.Results.List {
		count := len(field.Names)
		if count == 0 {
			count = 1
		}
		for j := 0; j < count; j++ {
			if i == len(original.Results.List)-1 && j == count-1 {
				break
			}
			value := r.nameFor("zero")
			prefix = append(prefix, variable(value,
				r.temporaryTypeAt(r.input.source.inventory.Checked.TypesInfo.TypeOf(field.Type), field.Type,
					original.End()), nil))
			r.functionState.frame.values = append(r.functionState.frame.values, value)
		}
	}
	return prefix
}

func (r *rewrite) propagatedCall(marker *ast.CallExpr, operation Propagation, discardSuccess bool) expressionResult {
	call := marker.Args[0]
	evaluation := r.expression(call)
	saved := r.saveResults(evaluation.Values[0], discardSuccess, true)
	statements := append(evaluation.Statements, saved.Statements...)
	if len(saved.Values) == 0 {
		panic("propagation requires an error result")
	}

	signature := r.frameSignature()
	returnErrorType := signature.Results().At(signature.Results().Len() - 1).Type()
	if _, ok := returnErrorType.Underlying().(*types.Interface); !ok || !types.AssignableTo(returnErrorType,
		types.Universe.Lookup("error").Type()) {
		panic("propagation requires a final error interface return")
	}
	if !r.functionState.frame.prepared {
		r.functionState.frame.prefix = r.prepareReturns(r.functionState.frame.typ)
		r.functionState.frame.prepared = true
	}

	callError := saved.Values[len(saved.Values)-1]
	failureStatements, failed := r.callFailure(callError, call)
	statements = append(statements, failureStatements...)
	errorValue := r.propagatedError(callError.Expression, returnErrorType, operation, call)
	results := append(append([]ast.Expr(nil), r.functionState.frame.values...), errorValue)
	failure := &ast.IfStmt{Cond: failed, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: results}}}}
	statements = append(statements, failure)
	r.request.inserted = true
	r.request.lowered[marker] = true
	if discardSuccess {
		return expressionResult{Statements: statements}
	}
	return expressionResult{Statements: statements, Values: saved.Values[:len(saved.Values)-1]}
}

func (r *rewrite) frameSignature() *types.Signature {
	var signature *types.Signature
	switch function := r.functionState.frame.function.(type) {
	case *ast.FuncDecl:
		signature = r.input.source.inventory.Checked.TypesInfo.Defs[function.Name].Type().(*types.Signature)
	case *ast.FuncLit:
		signature = r.input.source.inventory.Checked.TypesInfo.TypeOf(function).(*types.Signature)
	}
	return signature
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
func (r *rewrite) callFailure(result expressionValue, call ast.Expr) ([]ast.Stmt, ast.Expr) {
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
