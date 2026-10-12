package lowering

import (
	"github.com/lucasavila00/tgo/internal/checker/types"
	"go/ast"
	"go/constant"
	"go/token"
)

// contextLiteral keeps an omitted literal type through its original container.
func (r *rewrite) contextLiteral(value, original ast.Expr) ast.Expr {
	literal, ok := original.(*ast.CompositeLit)
	if !ok || literal.Type != nil {
		return value
	}
	parent := r.input.source.inventory.Parents[original]
	if pair, ok := parent.(*ast.KeyValueExpr); ok {
		parent = r.input.source.inventory.Parents[pair]
	}
	container, ok := parent.(*ast.CompositeLit)
	if !ok {
		return value
	}
	typ := r.input.source.inventory.Checked.TypesInfo.TypeOf(container).Underlying()
	pointer := false
	if p, ok := typ.(*types.Pointer); ok {
		typ = p.Elem().Underlying()
		pointer = true
	}
	wrapper := &ast.CompositeLit{Type: container.Type}
	var access ast.Expr
	switch t := typ.(type) {
	case *types.Struct:
		name := ""
		for i, element := range container.Elts {
			if element == original {
				name = t.Field(i).Name()
				break
			}
			if pair, ok := element.(*ast.KeyValueExpr); ok && (pair.Value == original || pair.Key == original) {
				name = pair.Key.(*ast.Ident).Name
				break
			}
		}
		wrapper.Elts = []ast.Expr{&ast.KeyValueExpr{Key: ast.NewIdent(name), Value: value}}
		access = &ast.SelectorExpr{Sel: ast.NewIdent(name)}
	case *types.Array, *types.Slice:
		wrapper.Elts = []ast.Expr{value}
		access = &ast.IndexExpr{Index: integer(0)}
	case *types.Map:
		return r.contextMapLiteral(value, original, container, wrapper, pointer)
	default:
		return value
	}
	var wrapped ast.Expr = wrapper
	if container.Type == nil {
		wrapped = r.contextLiteral(wrapper, container)
	}
	if pointer {
		wrapped = &ast.StarExpr{X: wrapped}
	}
	wrapped = &ast.ParenExpr{X: wrapped}
	switch a := access.(type) {
	case *ast.SelectorExpr:
		a.X = wrapped
	case *ast.IndexExpr:
		a.X = wrapped
	}
	return access
}

func (r *rewrite) contextMapLiteral(value, original ast.Expr, container, wrapper *ast.CompositeLit, pointer bool) ast.Expr {
	pair := r.input.source.inventory.Parents[original].(*ast.KeyValueExpr)
	if pair.Key == original {
		r.prepareCollectionImports()
		empty := r.contextLiteral(&ast.CompositeLit{Type: container.Type}, container)
		if pointer {
			empty = &ast.StarExpr{X: empty}
		}
		values := &ast.CallExpr{Fun: &ast.SelectorExpr{X: r.file.mapAlias, Sel: ast.NewIdent("Values")}, Args: []ast.Expr{empty}}
		wrapper.Elts = []ast.Expr{&ast.KeyValueExpr{Key: value, Value: r.sequenceZero(values)}}
		var mapValue ast.Expr = wrapper
		if container.Type == nil {
			mapValue = r.contextLiteral(wrapper, container)
		}
		if pointer {
			mapValue = &ast.StarExpr{X: mapValue}
		}
		keys := &ast.CallExpr{Fun: &ast.SelectorExpr{X: r.file.mapAlias, Sel: ast.NewIdent("Keys")}, Args: []ast.Expr{mapValue}}
		r.prepareSliceImport()
		collected := &ast.CallExpr{Fun: &ast.SelectorExpr{X: r.file.sliceAlias, Sel: ast.NewIdent("Collect")}, Args: []ast.Expr{keys}}
		return &ast.IndexExpr{X: collected, Index: integer(0)}
	}
	key := r.zeroMapKey(container, pointer)
	wrapper.Elts = []ast.Expr{&ast.KeyValueExpr{Key: key, Value: value}}
	r.prepareCollectionImports()
	var mapValue ast.Expr = wrapper
	if container.Type == nil {
		mapValue = r.contextLiteral(wrapper, container)
	}
	if pointer {
		mapValue = &ast.StarExpr{X: mapValue}
	}
	values := &ast.CallExpr{Fun: &ast.SelectorExpr{X: r.file.mapAlias, Sel: ast.NewIdent("Values")}, Args: []ast.Expr{mapValue}}
	r.prepareSliceImport()
	collected := &ast.CallExpr{Fun: &ast.SelectorExpr{X: r.file.sliceAlias, Sel: ast.NewIdent("Collect")}, Args: []ast.Expr{values}}
	return &ast.IndexExpr{X: collected, Index: integer(0)}
}

// materializeKey constructs a struct or array key without hashing it early.
func (r *rewrite) materializeKey(value, original *ast.CompositeLit) ([]ast.Stmt, []ast.Expr) {
	var elements []ast.Expr
	for _, element := range value.Elts {
		if item, ok := element.(*ast.KeyValueExpr); ok {
			elements = append(elements, item.Value)
		} else {
			elements = append(elements, element)
		}
	}
	var statements []ast.Stmt
	var values []ast.Expr
	for _, element := range elements {
		if tv, ok := r.input.source.inventory.Checked.TypesInfo.Types[element]; ok && (tv.Value != nil || tv.IsNil()) {
			values = append(values, element)
			continue
		}
		saveStatements, refs := r.save(element)
		statements = append(statements, saveStatements...)
		values = append(values, refs[0])
	}
	key := r.name()
	r.recordTemporary(key, r.typeOf(original), original)
	statements = append(statements, variable(key, r.temporaryType(r.typeOf(original), original), nil))
	index := 0
	for i, element := range value.Elts {
		var place ast.Expr
		switch typ := r.input.source.inventory.Checked.TypesInfo.TypeOf(original).Underlying().(type) {
		case *types.Struct:
			name := typ.Field(i).Name()
			if item, ok := element.(*ast.KeyValueExpr); ok {
				name = item.Key.(*ast.Ident).Name
			}
			place = &ast.SelectorExpr{X: key, Sel: ast.NewIdent(name)}
		case *types.Array:
			var position ast.Expr = integer(index)
			if item, ok := element.(*ast.KeyValueExpr); ok {
				position = item.Key
				positionValue, _ := constant.Int64Val(r.input.source.inventory.Checked.TypesInfo.Types[item.Key].Value)
				index = int(positionValue)
			}
			place = &ast.IndexExpr{X: key, Index: position}
			index++
		}
		statements = append(statements, &ast.AssignStmt{Lhs: []ast.Expr{place}, Tok: token.ASSIGN, Rhs: []ast.Expr{values[i]}})
	}
	return statements, []ast.Expr{key}
}

func (r *rewrite) zeroMapKey(container *ast.CompositeLit, pointer bool) ast.Expr {
	r.prepareCollectionImports()
	empty := r.contextLiteral(&ast.CompositeLit{Type: container.Type}, container)
	if pointer {
		empty = &ast.StarExpr{X: empty}
	}
	keys := &ast.CallExpr{Fun: &ast.SelectorExpr{X: r.file.mapAlias, Sel: ast.NewIdent("Keys")}, Args: []ast.Expr{empty}}
	return r.sequenceZero(keys)
}

func (r *rewrite) prepareCollectionImports() {
	if r.file.mapAlias == nil {
		r.file.mapAlias = r.nameFor("maps")
		r.file.extraImports = append(r.file.extraImports, &ast.ImportSpec{Name: r.file.mapAlias, Path: &ast.BasicLit{Kind: token.STRING, Value: `"maps"`}})
	}
}

func (r *rewrite) prepareSliceImport() {
	if r.file.sliceAlias == nil {
		r.file.sliceAlias = r.nameFor("slices")
		r.file.extraImports = append(r.file.extraImports, &ast.ImportSpec{Name: r.file.sliceAlias, Path: &ast.BasicLit{Kind: token.STRING, Value: `"slices"`}})
	}
}
