package compilerv2

import (
	"go/ast"
	"go/token"
	"go/types"
	"strconv"
)

type returnFrame struct {
	function   ast.Node
	typ        *ast.FuncType
	outputType *ast.FuncType
	prefix     []ast.Stmt
	values     []ast.Expr
	nilError   ast.Expr
}

func (r *rewrite) function(node ast.Node, typ *ast.FuncType, body *ast.BlockStmt) (*ast.FuncType, *ast.BlockStmt) {
	parent := r.frame
	r.frame = &returnFrame{function: node, typ: typ}
	output := r.block(body)
	if r.frame.outputType != nil {
		typ = r.frame.outputType
		output.List = append(r.frame.prefix, output.List...)
	}
	if aliases := r.aliasStatements[node]; len(aliases) != 0 {
		output.List = append(aliases, output.List...)
	}
	r.frame = parent
	return typ, output
}

// prepareReturns obtains typed zero results without spelling their types in the body.
func (r *rewrite) prepareReturns(original *ast.FuncType) (*ast.FuncType, []ast.Stmt) {
	if original.Results == nil {
		panic("propagation requires an error return")
	}
	copy := *original
	results := *original.Results
	results.List = make([]*ast.Field, len(original.Results.List))
	var prefix []ast.Stmt
	for i, field := range original.Results.List {
		clone := *field
		clone.Names = append([]*ast.Ident(nil), field.Names...)
		if len(clone.Names) == 0 {
			clone.Names = []*ast.Ident{r.name()}
		}
		for j, name := range clone.Names {
			if name.Name == "_" {
				name = r.name()
				clone.Names[j] = name
			}
			value := r.name()
			prefix = append(prefix, &ast.AssignStmt{Lhs: []ast.Expr{value}, Tok: token.DEFINE, Rhs: []ast.Expr{name}})
			r.frame.values = append(r.frame.values, value)
		}
		results.List[i] = &clone
	}
	copy.Results = &results
	r.frame.nilError = r.name()
	prefix = append(prefix, &ast.AssignStmt{Lhs: []ast.Expr{r.frame.nilError}, Tok: token.DEFINE, Rhs: r.frame.values[len(r.frame.values)-1:]})
	return &copy, prefix
}

func (r *rewrite) propagatedCall(marker *ast.CallExpr, operation propagation) ([]ast.Stmt, []ast.Expr) {
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
		signature = r.source.Package.TypesInfo.Defs[fn.Name].Type().(*types.Signature)
	case *ast.FuncLit:
		signature = r.source.Package.TypesInfo.TypeOf(fn).(*types.Signature)
	}
	last := signature.Results().At(signature.Results().Len() - 1).Type()
	if _, ok := last.Underlying().(*types.Interface); !ok || !types.AssignableTo(last, types.Universe.Lookup("error").Type()) {
		panic("propagation requires a final error interface return")
	}
	if r.frame.outputType == nil {
		r.frame.outputType, r.frame.prefix = r.prepareReturns(r.frame.typ)
	}
	errorValue := r.frame.values[len(r.frame.values)-1]
	before = append(before, &ast.AssignStmt{Lhs: []ast.Expr{errorValue}, Tok: token.ASSIGN, Rhs: values[len(values)-1:]})
	var failure []ast.Stmt
	if operation.wrap {
		if !types.AssignableTo(types.Universe.Lookup("error").Type(), last) {
			panic("wrapped errors do not fit the function return type")
		}
		if r.formatAlias == nil {
			r.formatAlias = r.name()
			r.extraImports = append(r.extraImports, &ast.ImportSpec{Name: r.formatAlias, Path: &ast.BasicLit{Kind: token.STRING, Value: `"fmt"`}})
		}
		wrapped := &ast.CallExpr{Fun: &ast.SelectorExpr{X: r.formatAlias, Sel: ast.NewIdent("Errorf")}, Args: []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: strconv.Quote("%s: %w")}, &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(operation.callName)}, errorValue}}
		failure = append(failure, &ast.AssignStmt{Lhs: []ast.Expr{errorValue}, Tok: token.ASSIGN, Rhs: []ast.Expr{wrapped}})
	}
	failure = append(failure, &ast.ReturnStmt{Results: r.frame.values})
	before = append(before, &ast.IfStmt{Cond: &ast.BinaryExpr{X: errorValue, Op: token.NEQ, Y: r.frame.nilError}, Body: &ast.BlockStmt{List: failure}})
	r.inserted = true
	return before, values[:len(values)-1]
}
