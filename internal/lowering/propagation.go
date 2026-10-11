package lowering

import (
	"github.com/lucasavila00/tgo/internal/checker/types"
	"go/ast"
	"go/token"
	"strconv"
	"strings"
)

type returnFrame struct {
	function ast.Node
	typ      *ast.FuncType
	prepared bool
	prefix   []ast.Stmt
	values   []ast.Expr
}

func (r *rewrite) function(node ast.Node, typ *ast.FuncType, body *ast.BlockStmt) (*ast.FuncType, *ast.BlockStmt) {
	parent := r.frame
	r.frame = &returnFrame{function: node, typ: typ}
	output := r.block(body)
	if r.frame.prepared {
		output.List = append(r.frame.prefix, output.List...)
	}
	if r.source.inventory.Checked.TypesInfo.TGoReturnBlocks[body] {
		output = r.returnBlock(typ, output)
	}
	if aliases := r.aliasStatements[node]; len(aliases) != 0 {
		output.List = append(aliases, output.List...)
	}
	r.frame = parent
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
			value := r.name()
			prefix = append(prefix, variable(value, r.temporaryTypeAt(r.source.inventory.Checked.TypesInfo.TypeOf(field.Type), field.Type, original.End()), nil))
			r.frame.values = append(r.frame.values, value)
		}
	}
	return prefix
}

func (r *rewrite) propagatedCall(marker *ast.CallExpr, operation Propagation) ([]ast.Stmt, []ast.Expr) {
	raw := marker.Args[0]
	before, rewritten := r.expression(raw)
	saved, values := r.save(rewritten[0])
	before = append(before, saved...)
	if len(values) == 0 {
		panic("propagation requires an error result")
	}
	function := r.frame.function
	var signature *types.Signature
	switch fn := function.(type) {
	case *ast.FuncDecl:
		signature = r.source.inventory.Checked.TypesInfo.Defs[fn.Name].Type().(*types.Signature)
	case *ast.FuncLit:
		signature = r.source.inventory.Checked.TypesInfo.TypeOf(fn).(*types.Signature)
	}
	last := signature.Results().At(signature.Results().Len() - 1).Type()
	if _, ok := last.Underlying().(*types.Interface); !ok || !types.AssignableTo(last, types.Universe.Lookup("error").Type()) {
		panic("propagation requires a final error interface return")
	}
	if !r.frame.prepared {
		r.frame.prefix = r.prepareReturns(r.frame.typ)
		r.frame.prepared = true
	}
	callError := values[len(values)-1]
	check, failed := r.callFailure(callError, raw)
	before = append(before, check...)
	var errorValue ast.Expr = callError
	if operation.Wrap {
		if !types.AssignableTo(types.Universe.Lookup("error").Type(), last) {
			panic("wrapped errors do not fit the function return type")
		}
		if r.formatAlias == nil {
			r.formatAlias = r.nameFor("fmt")
			r.extraImports = append(r.extraImports, &ast.ImportSpec{Name: r.formatAlias, Path: &ast.BasicLit{Kind: token.STRING, Value: `"fmt"`}})
		}
		errorValue = &ast.CallExpr{Fun: &ast.SelectorExpr{X: r.formatAlias, Sel: ast.NewIdent("Errorf")}, Args: []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(strings.ReplaceAll(operation.CallName, "%", "%%") + ": %w")}, errorValue}}
	}
	results := append(append([]ast.Expr(nil), r.frame.values...), errorValue)
	failure := []ast.Stmt{&ast.ReturnStmt{Results: results}}
	before = append(before, &ast.IfStmt{Cond: failed, Body: &ast.BlockStmt{List: failure}})
	r.inserted = true
	r.lowered[marker] = true
	return before, values[:len(values)-1]
}

// callFailure checks the call result before conversion to the caller's return type.
func (r *rewrite) callFailure(value ast.Expr, call ast.Expr) ([]ast.Stmt, ast.Expr) {
	typ := r.typeOf(value)
	_, generic := types.Unalias(typ).(*types.TypeParam)
	nilOnly, nilCapable := false, false
	switch typ.Underlying().(type) {
	case *types.Slice, *types.Map, *types.Signature:
		nilOnly, nilCapable = true, true
	case *types.Pointer, *types.Chan:
		nilCapable = true
	case *types.Interface:
		nilCapable = !generic
	}
	if generic && types.AssignableTo(types.Typ[types.UntypedNil], typ) {
		nilCapable = true
	}
	if nilCapable {
		scope := r.source.inventory.Checked.Types.Scope().Innermost(call.Pos())
		_, resolved := scope.LookupParent("nil", call.Pos())
		if resolved == types.Universe.Lookup("nil") {
			return nil, &ast.BinaryExpr{X: value, Op: token.NEQ, Y: ast.NewIdent("nil")}
		}
	}
	if generic || nilOnly {
		if r.reflectAlias == nil {
			r.reflectAlias = r.nameFor("reflect")
			r.extraImports = append(r.extraImports, &ast.ImportSpec{Name: r.reflectAlias, Path: &ast.BasicLit{Kind: token.STRING, Value: `"reflect"`}})
		}
		reflected, failed := r.name(), r.name()
		valueOf := &ast.CallExpr{Fun: &ast.SelectorExpr{X: r.reflectAlias, Sel: ast.NewIdent("ValueOf")}, Args: []ast.Expr{&ast.UnaryExpr{Op: token.AND, X: value}}}
		ref := &ast.CallExpr{Fun: &ast.SelectorExpr{X: valueOf, Sel: ast.NewIdent("Elem")}}
		var kinds []ast.Expr
		for _, kind := range []string{"Chan", "Func", "Interface", "Map", "Pointer", "Slice"} {
			kinds = append(kinds, &ast.SelectorExpr{X: r.reflectAlias, Sel: ast.NewIdent(kind)})
		}
		isNil := &ast.CallExpr{Fun: &ast.SelectorExpr{X: reflected, Sel: ast.NewIdent("IsNil")}}
		return []ast.Stmt{
			variable(reflected, &ast.SelectorExpr{X: r.reflectAlias, Sel: ast.NewIdent("Value")}, ref),
			variable(failed, r.temporaryType(types.Typ[types.Bool], call), boolean(true)),
			&ast.SwitchStmt{Tag: &ast.CallExpr{Fun: &ast.SelectorExpr{X: reflected, Sel: ast.NewIdent("Kind")}}, Body: &ast.BlockStmt{List: []ast.Stmt{
				&ast.CaseClause{List: kinds, Body: []ast.Stmt{&ast.AssignStmt{Lhs: []ast.Expr{failed}, Tok: token.ASSIGN, Rhs: []ast.Expr{&ast.UnaryExpr{Op: token.NOT, X: isNil}}}}},
			}}},
		}, failed
	}
	switch typ.Underlying().(type) {
	case *types.Pointer, *types.Chan, *types.Interface:
		zero := r.name()
		return []ast.Stmt{variable(zero, r.temporaryType(typ, call), nil)}, &ast.BinaryExpr{X: value, Op: token.NEQ, Y: zero}
	default:
		return nil, boolean(true)
	}
}
