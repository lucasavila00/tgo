package shared

import (
	"github.com/lucasavila00/tgo/internal/ast"
	"github.com/lucasavila00/tgo/internal/checker/types"
	"go/token"
)

const (
	arrayStorage = 1
	otherStorage = 2
)

// storageKinds classifies possible types as array or other storage, including generic type constraints.
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

// parameterAlias reserves a type-parameter alias and adds its declaration to the owning function.
// It reuses the alias for later generated declarations in the same file.
func (r *Context) parameterAlias(parameter *types.TypeParam) *ast.Ident {
	if alias := r.file.parameterAliases[parameter]; alias != nil {
		return alias
	}
	if r.file.parameterAliases == nil {
		r.file.parameterAliases = make(map[*types.TypeParam]*ast.Ident)
		r.file.aliasStatements = make(map[ast.Node][]ast.Stmt)
	}
	var owner ast.Node
	for id, object := range r.input.Inventory.Checked.TypesInfo.Defs {
		if object != parameter.Obj() {
			continue
		}
		for node := r.input.Inventory.Parents[id]; node != nil; node = r.input.Inventory.Parents[node] {
			if _, ok := node.(*ast.FuncDecl); ok {
				owner = node
				break
			}
		}
	}
	if owner == nil {
		panic("type parameter has no function declaration")
	}
	alias := r.Name()
	typeSpec := &ast.TypeSpec{Name: alias, Assign: 1, Type: ast.NewIdent(parameter.Obj().Name())}
	declaration := &ast.DeclStmt{Decl: &ast.GenDecl{Tok: token.TYPE, Specs: []ast.Spec{typeSpec}}}
	r.file.parameterAliases[parameter] = alias
	r.file.aliasStatements[owner] = append(r.file.aliasStatements[owner], declaration)
	return alias
}

// ReflectPackage returns the reflect import for the current file.
func (r *Context) ReflectPackage() *ast.Ident {
	if r.file.reflectAlias == nil {
		r.file.reflectAlias = r.NameFor("reflect")
		r.file.extraImports = append(r.file.extraImports, &ast.ImportSpec{Name: r.file.reflectAlias,
			Path: &ast.BasicLit{Kind: token.STRING, Value: `"reflect"`}})
	}
	return r.file.reflectAlias
}

// genericBase keeps array storage and saves slice or map headers.
func (r *Context) genericBase(expr ast.Expr) ([]ast.Stmt, ast.Expr) {
	parameter := types.Unalias(r.input.Inventory.Checked.TypesInfo.TypeOf(expr)).(*types.TypeParam)
	typeAlias := r.parameterAlias(parameter)
	reflectPackage := r.ReflectPackage()
	before, place := r.place(expr)
	isArray := r.Name()
	savedHeader := r.Name()
	typeFor := &ast.CallExpr{Fun: &ast.IndexExpr{X: &ast.SelectorExpr{X: reflectPackage,
		Sel: ast.NewIdent("TypeFor")}, Index: typeAlias}}
	kind := &ast.CallExpr{Fun: &ast.SelectorExpr{X: typeFor, Sel: ast.NewIdent("Kind")}}
	isArrayValue := &ast.BinaryExpr{X: kind, Op: token.EQL, Y: &ast.SelectorExpr{X: reflectPackage,
		Sel: ast.NewIdent("Array")}}
	before = append(before, Variable(isArray, r.TemporaryType(types.Typ[types.Bool], expr), isArrayValue))
	savedHeaderSpec := &ast.ValueSpec{Names: []*ast.Ident{savedHeader}, Type: typeAlias}
	before = append(before, &ast.DeclStmt{Decl: &ast.GenDecl{Tok: token.VAR, Specs: []ast.Spec{savedHeaderSpec}}})
	saveHeader := &ast.AssignStmt{Lhs: []ast.Expr{savedHeader}, Tok: token.ASSIGN, Rhs: []ast.Expr{place}}
	before = append(before, &ast.IfStmt{Cond: &ast.UnaryExpr{Op: token.NOT, X: isArray},
		Body: &ast.BlockStmt{List: []ast.Stmt{saveHeader}}})
	getter := genericBaseGetter(typeAlias, isArray, place, savedHeader)
	base := &ast.ParenExpr{X: &ast.StarExpr{X: &ast.CallExpr{Fun: getter}}}
	if r.functionState.deferredBases == nil {
		r.functionState.deferredBases = make(map[ast.Expr]bool)
	}
	r.functionState.deferredBases[base] = true
	return before, base
}

// genericBaseGetter builds a getter that returns original array storage or the saved slice or map header.
func genericBaseGetter(typeAlias, isArray *ast.Ident, place ast.Expr, savedHeader *ast.Ident) *ast.FuncLit {
	arrayAddress := &ast.UnaryExpr{Op: token.AND, X: place}
	returnArray := &ast.ReturnStmt{Results: []ast.Expr{arrayAddress}}
	arrayCase := &ast.IfStmt{Cond: isArray, Body: &ast.BlockStmt{List: []ast.Stmt{returnArray}}}
	headerAddress := &ast.UnaryExpr{Op: token.AND, X: savedHeader}
	returnHeader := &ast.ReturnStmt{Results: []ast.Expr{headerAddress}}
	returnType := &ast.StarExpr{X: typeAlias}
	functionType := &ast.FuncType{Params: &ast.FieldList{},
		Results: &ast.FieldList{List: []*ast.Field{{Type: returnType}}}}
	return &ast.FuncLit{Type: functionType, Body: &ast.BlockStmt{List: []ast.Stmt{arrayCase, returnHeader}}}
}

// deferredPlace reports whether destination syntax contains a generated storage getter whose check occurs at the store.
func (r *Context) deferredPlace(expr ast.Expr) bool {
	found := false
	ast.Inspect(expr, func(node ast.Node) bool {
		if expression, ok := node.(ast.Expr); ok && r.functionState.deferredBases[expression] {
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

// orderedStores builds separate stores in source order so each destination check occurs at its own store.
func (r *Context) orderedStores(lhs, rhs []ast.Expr, tok token.Token) []ast.Stmt {
	var output []ast.Stmt
	for i, place := range lhs {
		output = append(output, &ast.AssignStmt{Lhs: []ast.Expr{place}, Tok: tok, Rhs: []ast.Expr{rhs[i]}})
	}
	return output
}
