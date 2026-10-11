package lowering

import (
	"go/ast"
	"go/token"
	"go/types"
)

const (
	arrayStorage = 1
	otherStorage = 2
)

func storageKinds(typ types.Type) int {
	typ = types.Unalias(typ)
	if parameter, ok := typ.(*types.TypeParam); ok {
		return storageKinds(parameter.Constraint())
	}
	switch t := typ.Underlying().(type) {
	case *types.Array:
		return arrayStorage
	case *types.Interface:
		kinds := arrayStorage | otherStorage
		for i := 0; i < t.NumEmbeddeds(); i++ {
			kinds &= storageKinds(t.EmbeddedType(i))
		}
		return kinds
	case *types.Union:
		kinds := 0
		for i := 0; i < t.Len(); i++ {
			kinds |= storageKinds(t.Term(i).Type())
		}
		return kinds
	default:
		return otherStorage
	}
}

func (r *rewrite) parameterAlias(parameter *types.TypeParam) *ast.Ident {
	if alias := r.parameterAliases[parameter]; alias != nil {
		return alias
	}
	if r.parameterAliases == nil {
		r.parameterAliases = make(map[*types.TypeParam]*ast.Ident)
		r.aliasStatements = make(map[ast.Node][]ast.Stmt)
	}
	var owner ast.Node
	for id, object := range r.source.Package.TypesInfo.Defs {
		if object != parameter.Obj() {
			continue
		}
		for node := r.source.parents[id]; node != nil; node = r.source.parents[node] {
			if _, ok := node.(*ast.FuncDecl); ok {
				owner = node
				break
			}
		}
	}
	if owner == nil {
		panic("type parameter has no function declaration")
	}
	alias := r.name()
	declaration := &ast.DeclStmt{Decl: &ast.GenDecl{Tok: token.TYPE, Specs: []ast.Spec{&ast.TypeSpec{Name: alias, Assign: 1, Type: ast.NewIdent(parameter.Obj().Name())}}}}
	r.parameterAliases[parameter] = alias
	r.aliasStatements[owner] = append(r.aliasStatements[owner], declaration)
	return alias
}

// genericBase keeps array storage and saves slice or map headers.
func (r *rewrite) genericBase(expr ast.Expr) ([]ast.Stmt, ast.Expr) {
	parameter := types.Unalias(r.source.Package.TypesInfo.TypeOf(expr)).(*types.TypeParam)
	typ := r.parameterAlias(parameter)
	if r.reflectAlias == nil {
		r.reflectAlias = r.name()
		r.extraImports = append(r.extraImports, &ast.ImportSpec{Name: r.reflectAlias, Path: &ast.BasicLit{Kind: token.STRING, Value: `"reflect"`}})
	}
	before, place := r.place(expr)
	array, saved := r.name(), r.name()
	typeFor := &ast.CallExpr{Fun: &ast.IndexExpr{X: &ast.SelectorExpr{X: r.reflectAlias, Sel: ast.NewIdent("TypeFor")}, Index: typ}}
	kind := &ast.CallExpr{Fun: &ast.SelectorExpr{X: typeFor, Sel: ast.NewIdent("Kind")}}
	before = append(before,
		&ast.AssignStmt{Lhs: []ast.Expr{array}, Tok: token.DEFINE, Rhs: []ast.Expr{&ast.BinaryExpr{X: kind, Op: token.EQL, Y: &ast.SelectorExpr{X: r.reflectAlias, Sel: ast.NewIdent("Array")}}}},
		&ast.DeclStmt{Decl: &ast.GenDecl{Tok: token.VAR, Specs: []ast.Spec{&ast.ValueSpec{Names: []*ast.Ident{saved}, Type: typ}}}},
		&ast.IfStmt{Cond: &ast.UnaryExpr{Op: token.NOT, X: array}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.AssignStmt{Lhs: []ast.Expr{saved}, Tok: token.ASSIGN, Rhs: []ast.Expr{place}}}}},
	)
	getter := &ast.FuncLit{Type: &ast.FuncType{Params: &ast.FieldList{}, Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: typ}}}}}, Body: &ast.BlockStmt{List: []ast.Stmt{
		&ast.IfStmt{Cond: array, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{&ast.UnaryExpr{Op: token.AND, X: place}}}}}},
		&ast.ReturnStmt{Results: []ast.Expr{&ast.UnaryExpr{Op: token.AND, X: saved}}},
	}}}
	base := &ast.ParenExpr{X: &ast.StarExpr{X: &ast.CallExpr{Fun: getter}}}
	if r.deferredBases == nil {
		r.deferredBases = make(map[ast.Expr]bool)
	}
	r.deferredBases[base] = true
	return before, base
}

func (r *rewrite) deferredPlace(expr ast.Expr) bool {
	found := false
	ast.Inspect(expr, func(node ast.Node) bool {
		if expression, ok := node.(ast.Expr); ok && r.deferredBases[expression] {
			found = true
			return false
		}
		if _, closure := node.(*ast.FuncLit); closure {
			return false
		}
		return !found
	})
	return found
}

func (r *rewrite) orderedStores(lhs, rhs []ast.Expr, tok token.Token) []ast.Stmt {
	var output []ast.Stmt
	for i, place := range lhs {
		output = append(output, &ast.AssignStmt{Lhs: []ast.Expr{place}, Tok: tok, Rhs: []ast.Expr{rhs[i]}})
	}
	return output
}
