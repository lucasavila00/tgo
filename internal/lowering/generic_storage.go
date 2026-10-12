package lowering

import (
	"github.com/lucasavila00/tgo/internal/checker/types"
	"go/ast"
	"go/token"
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
	for id, object := range r.source.inventory.Checked.TypesInfo.Defs {
		if object != parameter.Obj() {
			continue
		}
		for node := r.source.inventory.Parents[id]; node != nil; node = r.source.inventory.Parents[node] {
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
	typeSpec := &ast.TypeSpec{Name: alias, Assign: 1, Type: ast.NewIdent(parameter.Obj().Name())}
	declaration := &ast.DeclStmt{Decl: &ast.GenDecl{Tok: token.TYPE, Specs: []ast.Spec{typeSpec}}}
	r.parameterAliases[parameter] = alias
	r.aliasStatements[owner] = append(r.aliasStatements[owner], declaration)
	return alias
}

func (r *rewrite) reflectPackage() *ast.Ident {
	if r.reflectAlias == nil {
		r.reflectAlias = r.nameFor("reflect")
		r.extraImports = append(r.extraImports, &ast.ImportSpec{Name: r.reflectAlias, Path: &ast.BasicLit{Kind: token.STRING, Value: `"reflect"`}})
	}
	return r.reflectAlias
}

// genericBase keeps array storage and saves slice or map headers.
func (r *rewrite) genericBase(expr ast.Expr) ([]ast.Stmt, ast.Expr) {
	parameter := types.Unalias(r.source.inventory.Checked.TypesInfo.TypeOf(expr)).(*types.TypeParam)
	typeAlias := r.parameterAlias(parameter)
	reflectPackage := r.reflectPackage()
	before, place := r.place(expr)
	isArray := r.name()
	savedHeader := r.name()
	typeFor := &ast.CallExpr{Fun: &ast.IndexExpr{X: &ast.SelectorExpr{X: reflectPackage, Sel: ast.NewIdent("TypeFor")}, Index: typeAlias}}
	kind := &ast.CallExpr{Fun: &ast.SelectorExpr{X: typeFor, Sel: ast.NewIdent("Kind")}}
	isArrayValue := &ast.BinaryExpr{X: kind, Op: token.EQL, Y: &ast.SelectorExpr{X: reflectPackage, Sel: ast.NewIdent("Array")}}
	before = append(before, variable(isArray, r.temporaryType(types.Typ[types.Bool], expr), isArrayValue))
	savedHeaderSpec := &ast.ValueSpec{Names: []*ast.Ident{savedHeader}, Type: typeAlias}
	before = append(before, &ast.DeclStmt{Decl: &ast.GenDecl{Tok: token.VAR, Specs: []ast.Spec{savedHeaderSpec}}})
	saveHeader := &ast.AssignStmt{Lhs: []ast.Expr{savedHeader}, Tok: token.ASSIGN, Rhs: []ast.Expr{place}}
	before = append(before, &ast.IfStmt{Cond: &ast.UnaryExpr{Op: token.NOT, X: isArray}, Body: &ast.BlockStmt{List: []ast.Stmt{saveHeader}}})
	getter := genericBaseGetter(typeAlias, isArray, place, savedHeader)
	base := &ast.ParenExpr{X: &ast.StarExpr{X: &ast.CallExpr{Fun: getter}}}
	if r.deferredBases == nil {
		r.deferredBases = make(map[ast.Expr]bool)
	}
	r.deferredBases[base] = true
	return before, base
}

func genericBaseGetter(typeAlias, isArray *ast.Ident, place ast.Expr, savedHeader *ast.Ident) *ast.FuncLit {
	arrayAddress := &ast.UnaryExpr{Op: token.AND, X: place}
	returnArray := &ast.ReturnStmt{Results: []ast.Expr{arrayAddress}}
	arrayCase := &ast.IfStmt{Cond: isArray, Body: &ast.BlockStmt{List: []ast.Stmt{returnArray}}}
	headerAddress := &ast.UnaryExpr{Op: token.AND, X: savedHeader}
	returnHeader := &ast.ReturnStmt{Results: []ast.Expr{headerAddress}}
	returnType := &ast.StarExpr{X: typeAlias}
	functionType := &ast.FuncType{Params: &ast.FieldList{}, Results: &ast.FieldList{List: []*ast.Field{{Type: returnType}}}}
	return &ast.FuncLit{Type: functionType, Body: &ast.BlockStmt{List: []ast.Stmt{arrayCase, returnHeader}}}
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
