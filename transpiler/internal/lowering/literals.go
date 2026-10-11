package lowering

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
)

// contextLiteral keeps an omitted literal type through its original container.
func (r *rewrite) contextLiteral(value, original ast.Expr) ast.Expr {
	literal, ok := original.(*ast.CompositeLit)
	if !ok || literal.Type != nil {
		return value
	}
	parent := r.source.parents[original]
	if pair, ok := parent.(*ast.KeyValueExpr); ok {
		parent = r.source.parents[pair]
	}
	container, ok := parent.(*ast.CompositeLit)
	if !ok {
		return value
	}
	typ := r.source.Package.TypesInfo.TypeOf(container).Underlying()
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
		pair := r.source.parents[original].(*ast.KeyValueExpr)
		if pair.Key == original {
			r.prepareCollectionImports()
			empty := r.contextLiteral(&ast.CompositeLit{Type: container.Type}, container)
			if pointer {
				empty = &ast.StarExpr{X: empty}
			}
			values := &ast.CallExpr{Fun: &ast.SelectorExpr{X: r.mapAlias, Sel: ast.NewIdent("Values")}, Args: []ast.Expr{empty}}
			wrapper.Elts = []ast.Expr{&ast.KeyValueExpr{Key: value, Value: r.sequenceZero(values)}}
			var mapValue ast.Expr = wrapper
			if container.Type == nil {
				mapValue = r.contextLiteral(wrapper, container)
			}
			if pointer {
				mapValue = &ast.StarExpr{X: mapValue}
			}
			keys := &ast.CallExpr{Fun: &ast.SelectorExpr{X: r.mapAlias, Sel: ast.NewIdent("Keys")}, Args: []ast.Expr{mapValue}}
			r.prepareSliceImport()
			collected := &ast.CallExpr{Fun: &ast.SelectorExpr{X: r.sliceAlias, Sel: ast.NewIdent("Collect")}, Args: []ast.Expr{keys}}
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
		values := &ast.CallExpr{Fun: &ast.SelectorExpr{X: r.mapAlias, Sel: ast.NewIdent("Values")}, Args: []ast.Expr{mapValue}}
		r.prepareSliceImport()
		collected := &ast.CallExpr{Fun: &ast.SelectorExpr{X: r.sliceAlias, Sel: ast.NewIdent("Collect")}, Args: []ast.Expr{values}}
		return &ast.IndexExpr{X: collected, Index: integer(0)}
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

// materializeKey constructs a struct or array key without hashing it early.
func (r *rewrite) materializeKey(value, original *ast.CompositeLit) ([]ast.Stmt, []ast.Expr) {
	pair := r.source.parents[original].(*ast.KeyValueExpr)
	container := r.source.parents[pair].(*ast.CompositeLit)
	var elements []ast.Expr
	for _, element := range value.Elts {
		if item, ok := element.(*ast.KeyValueExpr); ok {
			elements = append(elements, item.Value)
		} else {
			elements = append(elements, element)
		}
	}
	var before []ast.Stmt
	var values []ast.Expr
	for _, element := range elements {
		if tv, ok := r.source.Package.TypesInfo.Types[element]; ok && (tv.Value != nil || tv.IsNil()) {
			values = append(values, element)
			continue
		}
		setup, refs := r.save(element)
		before = append(before, setup...)
		values = append(values, refs[0])
	}
	key := r.name()
	before = append(before, &ast.AssignStmt{Lhs: []ast.Expr{key}, Tok: token.DEFINE, Rhs: []ast.Expr{r.zeroMapKey(container, isPointer(r.source.Package.TypesInfo.TypeOf(container)))}})
	index := 0
	for i, element := range value.Elts {
		var place ast.Expr
		switch typ := r.source.Package.TypesInfo.TypeOf(original).Underlying().(type) {
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
				positionValue, _ := constant.Int64Val(r.source.Package.TypesInfo.Types[item.Key].Value)
				index = int(positionValue)
			}
			place = &ast.IndexExpr{X: key, Index: position}
			index++
		}
		before = append(before, &ast.AssignStmt{Lhs: []ast.Expr{place}, Tok: token.ASSIGN, Rhs: []ast.Expr{values[i]}})
	}
	return before, []ast.Expr{key}
}

func (r *rewrite) zeroMapKey(container *ast.CompositeLit, pointer bool) ast.Expr {
	r.prepareCollectionImports()
	empty := r.contextLiteral(&ast.CompositeLit{Type: container.Type}, container)
	if pointer {
		empty = &ast.StarExpr{X: empty}
	}
	keys := &ast.CallExpr{Fun: &ast.SelectorExpr{X: r.mapAlias, Sel: ast.NewIdent("Keys")}, Args: []ast.Expr{empty}}
	return r.sequenceZero(keys)
}

func (r *rewrite) prepareCollectionImports() {
	if r.mapAlias == nil {
		r.mapAlias = r.name()
		r.extraImports = append(r.extraImports, &ast.ImportSpec{Name: r.mapAlias, Path: &ast.BasicLit{Kind: token.STRING, Value: `"maps"`}})
	}
}

func (r *rewrite) prepareSliceImport() {
	if r.sliceAlias == nil {
		r.sliceAlias = r.name()
		r.extraImports = append(r.extraImports, &ast.ImportSpec{Name: r.sliceAlias, Path: &ast.BasicLit{Kind: token.STRING, Value: `"slices"`}})
	}
}

func isPointer(typ types.Type) bool { _, ok := typ.Underlying().(*types.Pointer); return ok }
